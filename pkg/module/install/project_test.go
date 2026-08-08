package install

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscoverInstallProjectDoesNotTreatGoFailureAsMissingDependency(t *testing.T) {
	want := errors.New("module graph unavailable")
	root := t.TempDir()
	goModPath := filepath.Join(root, "go.mod")
	runner := &scriptedGoRunner{responses: map[string]scriptedGoResponse{
		"env GOMOD":         {output: goModPath},
		"list -m -json":     {output: `{"Path":"example.com/app","Main":true}`},
		"list -m -json all": {err: want},
	}}

	_, err := discoverInstallProject(context.Background(), runner, root)
	var missing *MissingDependencyError
	if !errors.Is(err, want) || errors.As(err, &missing) || !strings.Contains(err.Error(), "inspect project module graph") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDiscoverInstallProjectAllowsMissingFerret(t *testing.T) {
	root := t.TempDir()
	goModPath := filepath.Join(root, "go.mod")
	runner := &scriptedGoRunner{responses: map[string]scriptedGoResponse{
		"env GOMOD":         {output: goModPath},
		"list -m -json":     {output: `{"Path":"example.com/app","Main":true}`},
		"list -m -json all": {output: `{"Path":"example.com/app","Main":true}`},
	}}

	project, err := discoverInstallProject(context.Background(), runner, root)
	if err != nil {
		t.Fatal(err)
	}
	if project.FerretVersion != "" {
		t.Fatalf("unexpected Ferret version: %q", project.FerretVersion)
	}
}
