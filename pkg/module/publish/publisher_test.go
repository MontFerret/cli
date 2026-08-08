package publish

import (
	"context"
	"errors"
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
