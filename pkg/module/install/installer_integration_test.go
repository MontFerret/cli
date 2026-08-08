package install

import (
	"archive/zip"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	barnregistry "github.com/MontFerret/barn/pkg/registry"
)

func TestInstallerUpdatesProjectAndIsIdempotent(t *testing.T) {
	project, proxy := newInstallerTestProject(t)
	modulePath := "example.com/ferret/archive"
	writeProxyModule(t, proxy, modulePath, "v1.0.0", validInstallModuleSource())
	configureInstallerProxy(t, proxy)

	registry := installTestRegistry("acme/archive", "1.0.0", modulePath, ">=2.0.0-alpha.43 <3.0.0")
	installer := New(registry, nil)

	result, err := installer.Install(context.Background(), Options{Reference: "acme/archive", Directory: project})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Changed || !result.SourceChanged || !result.DependenciesChanged || result.EditedFile != "main.go" {
		t.Fatalf("unexpected install result: %#v", result)
	}

	source := readInstallTestFile(t, filepath.Join(project, "main.go"))
	if !strings.Contains(source, "ferret.WithModules(") || !strings.Contains(source, `"example.com/ferret/archive"`) {
		t.Fatalf("module was not registered:\n%s", source)
	}
	goMod := readInstallTestFile(t, filepath.Join(project, "go.mod"))
	if !strings.Contains(goMod, "example.com/ferret/archive v1.0.0") {
		t.Fatalf("module dependency was not added:\n%s", goMod)
	}
	if _, err := os.Stat(filepath.Join(project, ".ferret")); !os.IsNotExist(err) {
		t.Fatalf("unexpected .ferret state directory: %v", err)
	}

	beforeSource := source
	beforeMod := goMod
	beforeSum := readInstallTestFile(t, filepath.Join(project, "go.sum"))
	result, err = installer.Install(context.Background(), Options{Reference: "acme/archive@1.0.0", Directory: project})
	if err != nil {
		t.Fatal(err)
	}
	if result.Changed {
		t.Fatalf("expected idempotent install, got %#v", result)
	}
	if got := readInstallTestFile(t, filepath.Join(project, "main.go")); got != beforeSource {
		t.Fatal("idempotent install changed source")
	}
	if got := readInstallTestFile(t, filepath.Join(project, "go.mod")); got != beforeMod {
		t.Fatal("idempotent install changed go.mod")
	}
	if got := readInstallTestFile(t, filepath.Join(project, "go.sum")); got != beforeSum {
		t.Fatal("idempotent install changed go.sum")
	}
}

func TestInstallerBuildFailurePreservesProject(t *testing.T) {
	project, proxy := newInstallerTestProject(t)
	modulePath := "example.com/ferret/configured"
	writeProxyModule(t, proxy, modulePath, "v1.0.0", `package configured

import "github.com/MontFerret/ferret/v2/pkg/module"

func New(required string) module.Module { return required }
`)
	configureInstallerProxy(t, proxy)

	mainPath := filepath.Join(project, "main.go")
	modPath := filepath.Join(project, "go.mod")
	beforeMain := readInstallTestFile(t, mainPath)
	beforeMod := readInstallTestFile(t, modPath)

	registry := installTestRegistry("acme/configured", "1.0.0", modulePath, ">=2.0.0-alpha.43 <3.0.0")
	_, err := New(registry, nil).Install(context.Background(), Options{
		Reference: "acme/configured@1.0.0",
		Directory: project,
	})
	if err == nil || !strings.Contains(err.Error(), "not enough arguments in call") {
		t.Fatalf("unexpected build error: %v", err)
	}
	if got := readInstallTestFile(t, mainPath); got != beforeMain {
		t.Fatal("failed install changed source")
	}
	if got := readInstallTestFile(t, modPath); got != beforeMod {
		t.Fatal("failed install changed go.mod")
	}
	if _, statErr := os.Stat(filepath.Join(project, "go.sum")); !os.IsNotExist(statErr) {
		t.Fatalf("failed install created go.sum: %v", statErr)
	}
}

func TestInstallerRejectsFerretVersionChange(t *testing.T) {
	project, proxy := newInstallerTestProject(t)
	modulePath := "example.com/ferret/future"
	writeProxyModuleWithFerret(t, proxy, modulePath, "v1.0.0", "v2.1.0", validInstallModuleSource())
	configureInstallerProxy(t, proxy)

	registry := installTestRegistry("acme/future", "1.0.0", modulePath, ">=2.0.0-alpha.43 <3.0.0")
	_, err := New(registry, nil).Install(context.Background(), Options{
		Reference: "acme/future",
		Directory: project,
	})
	if err == nil || !strings.Contains(err.Error(), "would change project Ferret") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInstallerRequiresProjectFerretDependency(t *testing.T) {
	project := t.TempDir()
	writeInstallTestFile(t, filepath.Join(project, "go.mod"), "module example.com/app\n\ngo 1.26.5\n")

	registry := installTestRegistry("acme/archive", "1.0.0", "example.com/ferret/archive", ">=2.0.0 <3.0.0")
	_, err := New(registry, nil).Install(context.Background(), Options{
		Reference: "acme/archive",
		Directory: project,
	})
	if err == nil || !strings.Contains(err.Error(), "project does not select github.com/MontFerret/ferret/v2") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func newInstallerTestProject(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	project := filepath.Join(root, "app")
	core := filepath.Join(root, "ferret")
	proxy := filepath.Join(root, "proxy")

	for _, directory := range []string{project, core, filepath.Join(core, "pkg", "module"), proxy} {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	writeInstallTestFile(t, filepath.Join(core, "go.mod"), "module github.com/MontFerret/ferret/v2\n\ngo 1.26.5\n")
	writeInstallTestFile(t, filepath.Join(core, "pkg", "module", "module.go"), "package module\n\ntype Module interface{}\n")
	writeInstallTestFile(t, filepath.Join(core, "ferret.go"), `package ferret

import "github.com/MontFerret/ferret/v2/pkg/module"

type Option func()
type Engine struct{}

func New(options ...Option) (*Engine, error) { return &Engine{}, nil }
func WithModules(modules ...module.Module) Option { return func() {} }
`)

	writeInstallTestFile(t, filepath.Join(project, "go.mod"), fmt.Sprintf(`module example.com/app

go 1.26.5

require github.com/MontFerret/ferret/v2 v2.0.0-alpha.44

replace github.com/MontFerret/ferret/v2 => %s
`, filepath.ToSlash(core)))
	writeInstallTestFile(t, filepath.Join(project, "main.go"), `package main

import "github.com/MontFerret/ferret/v2"

func main() {
	_, _ = ferret.New()
}
`)

	return project, proxy
}

func installTestRegistry(id, version, packagePath, constraint string) *fakeRegistry {
	return &fakeRegistry{
		modules: map[string]*barnregistry.Module{
			id: {ID: id, Versions: []barnregistry.VersionSummary{{Version: version}}},
		},
		versions: map[string]*barnregistry.Version{
			id + "@" + version: installTestVersion(id, version, constraint, packagePath),
		},
	}
}

func configureInstallerProxy(t *testing.T, proxy string) {
	t.Helper()
	moduleCache := filepath.Join(t.TempDir(), "go-mod-cache")
	t.Cleanup(func() {
		_ = filepath.Walk(moduleCache, func(path string, _ os.FileInfo, err error) error {
			if err == nil {
				_ = os.Chmod(path, 0o755)
			}
			return nil
		})
	})
	t.Setenv("GOPROXY", "file://"+filepath.ToSlash(proxy))
	t.Setenv("GOSUMDB", "off")
	t.Setenv("GONOSUMDB", "*")
	t.Setenv("GOWORK", "off")
	t.Setenv("GOCACHE", filepath.Join(t.TempDir(), "go-cache"))
	t.Setenv("GOMODCACHE", moduleCache)
}

func writeProxyModule(t *testing.T, proxy, modulePath, version, source string) {
	t.Helper()
	writeProxyModuleWithFerret(t, proxy, modulePath, version, "v2.0.0-alpha.44", source)
}

func writeProxyModuleWithFerret(t *testing.T, proxy, modulePath, version, ferretVersion, source string) {
	t.Helper()
	versionDir := filepath.Join(proxy, filepath.FromSlash(modulePath), "@v")
	if err := os.MkdirAll(versionDir, 0o755); err != nil {
		t.Fatal(err)
	}

	goMod := fmt.Sprintf("module %s\n\ngo 1.26.5\n\nrequire github.com/MontFerret/ferret/v2 %s\n", modulePath, ferretVersion)
	writeInstallTestFile(t, filepath.Join(versionDir, version+".mod"), goMod)
	writeInstallTestFile(t, filepath.Join(versionDir, version+".info"), fmt.Sprintf("{\"Version\":%q,\"Time\":\"2026-08-07T00:00:00Z\"}\n", version))
	writeInstallTestFile(t, filepath.Join(versionDir, "list"), version+"\n")

	archive, err := os.Create(filepath.Join(versionDir, version+".zip"))
	if err != nil {
		t.Fatal(err)
	}
	zipWriter := zip.NewWriter(archive)
	prefix := modulePath + "@" + version + "/"
	for name, content := range map[string]string{"go.mod": goMod, "module.go": source} {
		entry, err := zipWriter.Create(prefix + name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
}

func validInstallModuleSource() string {
	return `package archive

import "github.com/MontFerret/ferret/v2/pkg/module"

func New() module.Module { return struct{}{} }
`
}

func writeInstallTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readInstallTestFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	return string(data)
}
