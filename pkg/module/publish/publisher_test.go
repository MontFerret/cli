package publish

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	barnpublish "github.com/MontFerret/barn/pkg/publish"
	barnregistry "github.com/MontFerret/barn/pkg/registry"
)

func TestPublisherDerivesDefaultTag(t *testing.T) {
	for _, test := range []struct {
		name       string
		sourcePath string
		wantTag    string
	}{
		{name: "standalone", wantTag: "v1.2.3"},
		{name: "monorepo", sourcePath: "modules/widget", wantTag: "modules/widget/v1.2.3"},
	} {
		t.Run(test.name, func(t *testing.T) {
			directory := t.TempDir()
			writePublisherManifest(t, directory, test.sourcePath)

			client, err := barnregistry.NewClient(barnregistry.WithBaseURL("https://registry.example"))
			if err != nil {
				t.Fatal(err)
			}
			publisher := New(client)
			var captured barnpublish.Request
			wantResult := &barnpublish.Result{Kind: barnpublish.NewModule}
			publisher.prepare = func(_ context.Context, request barnpublish.Request) (*barnpublish.Result, error) {
				captured = request

				return wantResult, nil
			}

			result, err := publisher.Prepare(context.Background(), Options{Directory: directory})
			if err != nil {
				t.Fatal(err)
			}
			if result != wantResult || captured.Directory != directory || captured.Tag != test.wantTag || captured.Registry != client {
				t.Fatalf("unexpected Barn request: %#v", captured)
			}
		})
	}
}

func TestPublisherPreservesExplicitTagAndErrors(t *testing.T) {
	client, err := barnregistry.NewClient(barnregistry.WithBaseURL("https://registry.example"))
	if err != nil {
		t.Fatal(err)
	}
	publisher := New(client)
	want := errors.New("publication failed")
	var captured barnpublish.Request
	publisher.prepare = func(_ context.Context, request barnpublish.Request) (*barnpublish.Result, error) {
		captured = request

		return nil, want
	}

	_, err = publisher.Prepare(context.Background(), Options{Directory: "missing", Tag: "release/widget-1.2.3"})
	if !errors.Is(err, want) {
		t.Fatalf("expected delegated error, got %v", err)
	}
	if captured.Directory != "missing" || captured.Tag != "release/widget-1.2.3" || captured.Registry != client {
		t.Fatalf("unexpected explicit-tag request: %#v", captured)
	}
}

func TestPublisherValidatesManifestBeforeDerivingTag(t *testing.T) {
	directory := t.TempDir()
	publisher := New(nil)
	called := false
	publisher.prepare = func(context.Context, barnpublish.Request) (*barnpublish.Result, error) {
		called = true

		return nil, nil
	}

	if _, err := publisher.Prepare(context.Background(), Options{Directory: directory}); err == nil {
		t.Fatal("expected missing manifest to fail")
	}
	if called {
		t.Fatal("Barn publication ran after default-tag derivation failed")
	}
}

func TestPublisherPublishesPreparedRecords(t *testing.T) {
	directory := t.TempDir()
	writePublisherManifest(t, directory, "")
	wantPrepared := &barnpublish.Result{Kind: barnpublish.NewModule}
	submitter := &fakeSubmitter{submission: &Submission{URL: "https://github.com/MontFerret/barn/pull/123"}}
	publisher := New(nil, submitter)
	publisher.prepare = func(context.Context, barnpublish.Request) (*barnpublish.Result, error) {
		return wantPrepared, nil
	}

	result, err := publisher.Publish(context.Background(), Options{Directory: directory})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != StatusSubmitted || result.Prepared != wantPrepared || result.PullRequestURL != submitter.submission.URL {
		t.Fatalf("unexpected publication result: %#v", result)
	}
	if submitter.calls != 1 || submitter.prepared != wantPrepared {
		t.Fatalf("unexpected submitter call: calls=%d prepared=%#v", submitter.calls, submitter.prepared)
	}
}

func TestPublisherNonMutatingModesNeverSubmit(t *testing.T) {
	for _, mode := range []Mode{ModeDryRun, ModePrint} {
		t.Run(string(mode), func(t *testing.T) {
			directory := t.TempDir()
			writePublisherManifest(t, directory, "modules/widget")
			submitter := new(fakeSubmitter)
			publisher := New(nil, submitter)
			publisher.prepare = func(context.Context, barnpublish.Request) (*barnpublish.Result, error) {
				return &barnpublish.Result{Kind: barnpublish.NewVersion}, nil
			}

			result, err := publisher.Publish(context.Background(), Options{Directory: directory, Mode: mode})
			if err != nil {
				t.Fatal(err)
			}
			if result.Status != StatusReady || result.Tag != "modules/widget/v1.2.3" || submitter.calls != 0 {
				t.Fatalf("unexpected non-mutating result: result=%#v calls=%d", result, submitter.calls)
			}
		})
	}
}

func TestPublisherAlreadyPublishedIsSuccessfulNoOp(t *testing.T) {
	for _, mode := range []Mode{ModeSubmit, ModeDryRun, ModePrint} {
		t.Run(string(mode), func(t *testing.T) {
			directory := t.TempDir()
			writePublisherManifest(t, directory, "")
			submitter := new(fakeSubmitter)
			publisher := New(nil, submitter)
			publisher.prepare = func(context.Context, barnpublish.Request) (*barnpublish.Result, error) {
				return nil, fmt.Errorf("registry state: %w", barnpublish.ErrVersionAlreadyPublished)
			}

			result, err := publisher.Publish(context.Background(), Options{Directory: directory, Mode: mode})
			if err != nil {
				t.Fatal(err)
			}
			if result.Status != StatusAlreadyPublished || result.Module != "acme/widget" || result.Version != "1.2.3" || submitter.calls != 0 {
				t.Fatalf("unexpected already-published result: result=%#v calls=%d", result, submitter.calls)
			}
		})
	}
}

func TestPublisherReturnsPreparationAfterSubmissionFailure(t *testing.T) {
	directory := t.TempDir()
	writePublisherManifest(t, directory, "")
	wantErr := errors.New("create pull request")
	prepared := &barnpublish.Result{Kind: barnpublish.NewModule}
	publisher := New(nil, &fakeSubmitter{err: wantErr})
	publisher.prepare = func(context.Context, barnpublish.Request) (*barnpublish.Result, error) {
		return prepared, nil
	}

	result, err := publisher.Publish(context.Background(), Options{Directory: directory})
	if !errors.Is(err, wantErr) || result == nil || result.Status != StatusReady || result.Prepared != prepared {
		t.Fatalf("unexpected partial result: result=%#v err=%v", result, err)
	}
}

func writePublisherManifest(t *testing.T, directory, sourcePath string) {
	t.Helper()

	repository := "repository:\n  url: https://example.com/acme/widget.git\n"
	if sourcePath != "" {
		repository += "  directory: " + sourcePath + "\n"
	}
	manifest := `$schema: https://schemas.ferretlang.org/module/v1.json
name: acme/widget
namespace: WIDGET
version: 1.2.3
description: Widget support.
license: Apache-2.0
documentation: https://example.com/acme/widget
` + repository
	if err := os.WriteFile(filepath.Join(directory, "ferret.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
}
