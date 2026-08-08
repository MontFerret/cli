package scaffold

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	modulemanifest "github.com/MontFerret/specs/pkg/module"
)

func TestDefaultOptions(t *testing.T) {
	options, err := DefaultOptions("acme/sqlite")
	if err != nil {
		t.Fatal(err)
	}

	want := Options{
		Name:      "acme/sqlite",
		GoModule:  "github.com/acme/ferret-sqlite",
		Directory: "sqlite",
		Namespace: "sqlite",
	}
	if !reflect.DeepEqual(options, want) {
		t.Fatalf("unexpected defaults: %#v", options)
	}

	for _, name := range []string{"", "invalid", "Acme/sqlite"} {
		if _, err := DefaultOptions(name); err == nil {
			t.Fatalf("expected invalid name %q to fail", name)
		}
	}
}

func TestOptionsValidate(t *testing.T) {
	valid := Options{Name: "acme/sqlite", GoModule: "example.com/acme/sqlite"}
	if err := valid.Validate(); err != nil {
		t.Fatalf("expected defaults-compatible options to pass: %v", err)
	}

	for _, test := range []struct {
		name    string
		options Options
	}{
		{name: "missing name", options: Options{GoModule: "example.com/acme/sqlite"}},
		{name: "missing Go module", options: Options{Name: "acme/sqlite"}},
		{name: "invalid Go module", options: Options{Name: "acme/sqlite", GoModule: "not-a-module"}},
		{name: "invalid module name", options: Options{Name: "Acme/sqlite", GoModule: "example.com/acme/sqlite"}},
		{name: "invalid namespace", options: Options{Name: "acme/sqlite", GoModule: "example.com/acme/sqlite", Namespace: "INVALID-NAMESPACE"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := test.options.Validate(); err == nil {
				t.Fatalf("expected options to fail: %#v", test.options)
			}
		})
	}
}

func TestScaffolderCreatesValidatedProject(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)

	scaffolder := New(func() (Environment, error) {
		return Environment{GoVersion: "1.25.0", FerretVersion: "v2.0.0-alpha.44"}, nil
	})
	result, err := scaffolder.Create(context.Background(), Options{
		Name:     "db/sqlite",
		GoModule: "github.com/acme/ferret-sqlite",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Directory != filepath.Join(root, "sqlite") || result.Namespace != "sqlite" {
		t.Fatalf("unexpected result: %#v", result)
	}

	for _, name := range []string{"ferret.yaml", "go.mod", "module.go", "core/doc.go", "lib/doc.go", "README.md"} {
		if info, err := os.Stat(filepath.Join(result.Directory, filepath.FromSlash(name))); err != nil || !info.Mode().IsRegular() {
			t.Fatalf("expected generated file %s: %v", name, err)
		}
	}

	manifest, err := modulemanifest.LoadFile(filepath.Join(result.Directory, "ferret.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Name != "db/sqlite" || manifest.Namespace != "sqlite" || manifest.License != "LicenseRef-TODO" {
		t.Fatalf("unexpected manifest: %#v", manifest)
	}

	goMod, err := os.ReadFile(filepath.Join(result.Directory, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(goMod), "module github.com/acme/ferret-sqlite") || !strings.Contains(string(goMod), "github.com/MontFerret/ferret/v2 v2.0.0-alpha.44") {
		t.Fatalf("unexpected go.mod:\n%s", goMod)
	}
}

func TestScaffolderHonorsDirectoryAndNamespace(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "nested", "widget")
	scaffolder := New(func() (Environment, error) {
		return Environment{GoVersion: "1.26.5", FerretVersion: "v2.0.0-alpha.44"}, nil
	})

	result, err := scaffolder.Create(context.Background(), Options{
		Name:      "acme/widget",
		GoModule:  "example.com/acme/widget",
		Directory: destination,
		Namespace: "Acme::Widget",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Namespace != "Acme::Widget" {
		t.Fatalf("unexpected namespace: %q", result.Namespace)
	}
}

func TestScaffolderRejectsInvalidInputsAndExistingDestination(t *testing.T) {
	scaffolder := New(func() (Environment, error) {
		return Environment{GoVersion: "1.25.0", FerretVersion: "v2.0.0-alpha.44"}, nil
	})

	for _, options := range []Options{
		{Name: "invalid", GoModule: "example.com/module"},
		{Name: "acme/widget", GoModule: "not a module path"},
		{Name: "Acme/widget", GoModule: "example.com/module"},
		{Name: "acme/widget", GoModule: "example.com/module", Namespace: "INVALID-NAMESPACE"},
	} {
		if _, err := scaffolder.Create(context.Background(), options); err == nil {
			t.Fatalf("expected options to fail: %#v", options)
		}
	}

	destination := filepath.Join(t.TempDir(), "exists")
	if err := os.Mkdir(destination, 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := scaffolder.Create(context.Background(), Options{
		Name: "acme/widget", GoModule: "example.com/module", Directory: destination,
	})
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected existing destination error, got %v", err)
	}

	parentFile := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(parentFile, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	blockedDestination := filepath.Join(parentFile, "widget")
	_, err = scaffolder.Create(context.Background(), Options{
		Name: "acme/widget", GoModule: "example.com/module", Directory: blockedDestination,
	})
	if err == nil {
		t.Fatal("expected destination parent failure")
	}
	if _, statErr := os.Lstat(blockedDestination); statErr == nil {
		t.Fatal("expected no partial destination")
	}
}
