package install

import (
	"archive/zip"
	"context"
	"errors"
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

func TestInstallerReportsMissingProjectFerretDependency(t *testing.T) {
	project := t.TempDir()
	writeInstallTestFile(t, filepath.Join(project, "go.mod"), "module example.com/app\n\ngo 1.26.5\n")

	registry := installTestRegistry("acme/archive", "1.0.0", "example.com/ferret/archive", ">=2.0.0 <3.0.0")
	installer := New(registry, nil)
	installer.ferretVersion = func() (string, error) { return "v2.0.0-alpha.44", nil }
	_, err := installer.Install(context.Background(), Options{
		Reference: "acme/archive",
		Directory: project,
	})
	var missing *MissingDependencyError
	if !errors.As(err, &missing) || missing.Path != ferretCoreModulePath || missing.Version != "v2.0.0-alpha.44" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInstallerAddsApprovedFerretDependencyTransactionally(t *testing.T) {
	project, proxy := newInstallerTestProjectWithoutFerret(t)
	modulePath := "example.com/ferret/archive"
	writeProxyModule(t, proxy, modulePath, "v1.0.0", validInstallModuleSource())
	configureInstallerProxy(t, proxy)

	installer := New(installTestRegistry("acme/archive", "1.0.0", modulePath, ">=2.0.0-alpha.43 <3.0.0"), nil)
	installer.ferretVersion = func() (string, error) { return "v2.0.0-alpha.44", nil }

	result, err := installer.Install(context.Background(), Options{
		Reference:                  "acme/archive",
		Directory:                  project,
		InstallMissingDependencies: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Changed || !result.DependenciesChanged || !result.FerretDependencyAdded || result.ProjectFerret != "v2.0.0-alpha.44" {
		t.Fatalf("unexpected install result: %#v", result)
	}

	goMod := readInstallTestFile(t, filepath.Join(project, "go.mod"))
	if !strings.Contains(goMod, "github.com/MontFerret/ferret/v2 v2.0.0-alpha.44") || !strings.Contains(goMod, modulePath+" v1.0.0") {
		t.Fatalf("dependencies were not committed together:\n%s", goMod)
	}
	if source := readInstallTestFile(t, filepath.Join(project, "main.go")); !strings.Contains(source, `"`+modulePath+`"`) {
		t.Fatalf("module was not registered:\n%s", source)
	}

	result, err = installer.Install(context.Background(), Options{Reference: "acme/archive", Directory: project})
	if err != nil {
		t.Fatal(err)
	}
	if result.Changed || result.FerretDependencyAdded {
		t.Fatalf("expected idempotent install, got %#v", result)
	}
}

func TestInstallerApprovedFerretDependencyRollsBackOnBuildFailure(t *testing.T) {
	project, proxy := newInstallerTestProjectWithoutFerret(t)
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
	installer := New(installTestRegistry("acme/configured", "1.0.0", modulePath, ">=2.0.0-alpha.43 <3.0.0"), nil)
	installer.ferretVersion = func() (string, error) { return "v2.0.0-alpha.44", nil }

	_, err := installer.Install(context.Background(), Options{
		Reference:                  "acme/configured",
		Directory:                  project,
		InstallMissingDependencies: true,
	})
	if err == nil || !strings.Contains(err.Error(), "not enough arguments in call") {
		t.Fatalf("unexpected build error: %v", err)
	}
	if got := readInstallTestFile(t, mainPath); got != beforeMain {
		t.Fatal("failed install changed source")
	}
	if got := readInstallTestFile(t, modPath); got != beforeMod {
		t.Fatal("failed install committed Ferret dependency")
	}
	if _, statErr := os.Stat(filepath.Join(project, "go.sum")); !os.IsNotExist(statErr) {
		t.Fatalf("failed install created go.sum: %v", statErr)
	}
}

func TestInstallerScaffoldsEmptyProjectAndIsIdempotent(t *testing.T) {
	project, proxy := newEmptyInstallerTestProject(t)
	modulePath := "example.com/ferret/archive"
	writeProxyModule(t, proxy, modulePath, "v1.0.0", validInstallModuleSource())
	configureInstallerProxy(t, proxy)

	installer := New(installTestRegistry("acme/archive", "1.0.0", modulePath, ">=2.0.0-alpha.43 <3.0.0"), nil)
	installer.ferretVersion = func() (string, error) { return "v2.0.0-alpha.44", nil }

	_, err := installer.Install(context.Background(), Options{Reference: "acme/archive", Directory: project})
	var dependency *MissingDependencyError
	var composition *MissingCompositionError
	if !errors.As(err, &dependency) || !errors.As(err, &composition) {
		t.Fatalf("expected combined prerequisites, got %v", err)
	}
	if composition.File != "ferret.go" || composition.Package != "xproject" {
		t.Fatalf("unexpected scaffold proposal: %#v", composition)
	}

	result, err := installer.Install(context.Background(), Options{
		Reference:                  "acme/archive",
		Directory:                  project,
		InstallMissingDependencies: true,
		ScaffoldMissingComposition: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Changed || !result.SourceChanged || !result.DependenciesChanged || !result.FerretDependencyAdded || !result.CompositionScaffolded || result.EditedFile != "ferret.go" {
		t.Fatalf("unexpected scaffold result: %#v", result)
	}

	source := readInstallTestFile(t, filepath.Join(project, "ferret.go"))
	for _, expected := range []string{
		"package xproject",
		"func NewFerret(options ...ferret.Option) (*ferret.Engine, error)",
		"ferret.WithModules(",
		`"` + modulePath + `"`,
	} {
		if !strings.Contains(source, expected) {
			t.Fatalf("expected %q in scaffold:\n%s", expected, source)
		}
	}
	if strings.Contains(source, "func main(") {
		t.Fatalf("unexpected executable scaffold:\n%s", source)
	}

	result, err = installer.Install(context.Background(), Options{Reference: "acme/archive", Directory: project})
	if err != nil {
		t.Fatal(err)
	}
	if result.Changed || result.CompositionScaffolded || result.FerretDependencyAdded {
		t.Fatalf("expected idempotent install, got %#v", result)
	}
}

func TestInstallerScaffoldsSingleExistingPackage(t *testing.T) {
	project, proxy := newInstallerTestProject(t)
	writeInstallTestFile(t, filepath.Join(project, "main.go"), "package host\n\nfunc Existing() {}\n")
	modulePath := "example.com/ferret/archive"
	writeProxyModule(t, proxy, modulePath, "v1.0.0", validInstallModuleSource())
	configureInstallerProxy(t, proxy)

	result, err := New(
		installTestRegistry("acme/archive", "1.0.0", modulePath, ">=2.0.0-alpha.43 <3.0.0"),
		nil,
	).Install(context.Background(), Options{
		Reference:                  "acme/archive",
		Directory:                  project,
		ScaffoldMissingComposition: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.CompositionScaffolded || result.EditedFile != "ferret.go" {
		t.Fatalf("unexpected scaffold result: %#v", result)
	}
	if source := readInstallTestFile(t, filepath.Join(project, "ferret.go")); !strings.HasPrefix(source, "package host\n") {
		t.Fatalf("unexpected package scaffold:\n%s", source)
	}
}

func TestInstallerRejectsAmbiguousOrConflictingCompositionScaffold(t *testing.T) {
	t.Run("multiple packages", func(t *testing.T) {
		project, _ := newInstallerTestProject(t)
		writeInstallTestFile(t, filepath.Join(project, "main.go"), "package host\n")
		if err := os.MkdirAll(filepath.Join(project, "other"), 0o755); err != nil {
			t.Fatal(err)
		}
		writeInstallTestFile(t, filepath.Join(project, "other", "other.go"), "package other\n")

		_, err := New(installTestRegistry("acme/archive", "1.0.0", "example.com/archive", ">=2.0.0 <3.0.0"), nil).Install(
			context.Background(),
			Options{Reference: "acme/archive", Directory: project, ScaffoldMissingComposition: true},
		)
		if err == nil || !strings.Contains(err.Error(), "target package is ambiguous") || !strings.Contains(err.Error(), "example.com/app/other") {
			t.Fatalf("unexpected ambiguity error: %v", err)
		}
	})

	t.Run("ferret file", func(t *testing.T) {
		project, _ := newInstallerTestProject(t)
		writeInstallTestFile(t, filepath.Join(project, "main.go"), "package host\n")
		writeInstallTestFile(t, filepath.Join(project, "ferret.go"), "package host\n")

		_, err := New(installTestRegistry("acme/archive", "1.0.0", "example.com/archive", ">=2.0.0 <3.0.0"), nil).Install(
			context.Background(),
			Options{Reference: "acme/archive", Directory: project, ScaffoldMissingComposition: true},
		)
		if err == nil || !strings.Contains(err.Error(), "ferret.go already exists") {
			t.Fatalf("unexpected file conflict error: %v", err)
		}
	})

	t.Run("NewFerret declaration", func(t *testing.T) {
		project, _ := newInstallerTestProject(t)
		writeInstallTestFile(t, filepath.Join(project, "main.go"), "package host\n\nfunc NewFerret() {}\n")

		_, err := New(installTestRegistry("acme/archive", "1.0.0", "example.com/archive", ">=2.0.0 <3.0.0"), nil).Install(
			context.Background(),
			Options{Reference: "acme/archive", Directory: project, ScaffoldMissingComposition: true},
		)
		if err == nil || !strings.Contains(err.Error(), "NewFerret is already declared") {
			t.Fatalf("unexpected declaration conflict error: %v", err)
		}
	})
}

func TestInstallerScaffoldRollbackRemovesNewComposition(t *testing.T) {
	project, proxy := newEmptyInstallerTestProject(t)
	modulePath := "example.com/ferret/configured"
	writeProxyModule(t, proxy, modulePath, "v1.0.0", `package configured

import "github.com/MontFerret/ferret/v2/pkg/module"

func New(required string) module.Module { return required }
`)
	configureInstallerProxy(t, proxy)

	modPath := filepath.Join(project, "go.mod")
	beforeMod := readInstallTestFile(t, modPath)
	installer := New(installTestRegistry("acme/configured", "1.0.0", modulePath, ">=2.0.0-alpha.43 <3.0.0"), nil)
	installer.ferretVersion = func() (string, error) { return "v2.0.0-alpha.44", nil }

	_, err := installer.Install(context.Background(), Options{
		Reference:                  "acme/configured",
		Directory:                  project,
		InstallMissingDependencies: true,
		ScaffoldMissingComposition: true,
	})
	if err == nil || !strings.Contains(err.Error(), "not enough arguments in call") {
		t.Fatalf("unexpected build error: %v", err)
	}
	if got := readInstallTestFile(t, modPath); got != beforeMod {
		t.Fatal("failed scaffold changed go.mod")
	}
	if _, statErr := os.Stat(filepath.Join(project, "ferret.go")); !os.IsNotExist(statErr) {
		t.Fatalf("failed scaffold created ferret.go: %v", statErr)
	}
	if _, statErr := os.Stat(filepath.Join(project, "go.sum")); !os.IsNotExist(statErr) {
		t.Fatalf("failed scaffold created go.sum: %v", statErr)
	}
}

func newInstallerTestProject(t *testing.T) (string, string) {
	return newInstallerTestProjectFixture(t, "example.com/app", true, true)
}

func newInstallerTestProjectWithoutFerret(t *testing.T) (string, string) {
	return newInstallerTestProjectFixture(t, "example.com/app", false, true)
}

func newEmptyInstallerTestProject(t *testing.T) (string, string) {
	return newInstallerTestProjectFixture(t, "montferret.com/xproject", false, false)
}

func newInstallerTestProjectFixture(t *testing.T, modulePath string, includeFerret, includeComposition bool) (string, string) {
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

	requirement := ""
	if includeFerret {
		requirement = "require github.com/MontFerret/ferret/v2 v2.0.0-alpha.44\n\n"
	}
	writeInstallTestFile(t, filepath.Join(project, "go.mod"), fmt.Sprintf(`module %s

go 1.26.5

%s
replace github.com/MontFerret/ferret/v2 => %s
`, modulePath, requirement, filepath.ToSlash(core)))
	if includeComposition {
		writeInstallTestFile(t, filepath.Join(project, "main.go"), `package main

import "github.com/MontFerret/ferret/v2"

func main() {
	_, _ = ferret.New()
}
`)
	}

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
