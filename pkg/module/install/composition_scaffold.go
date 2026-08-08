package install

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

func planCompositionScaffold(project *projectInfo, packages []goPackageInfo) (*compositionScaffold, error) {
	var scaffold compositionScaffold

	switch len(packages) {
	case 0:
		if err := ensureProjectHasNoGoFiles(project.Root); err != nil {
			return nil, err
		}

		scaffold = compositionScaffold{
			Directory:   project.Root,
			PackageName: scaffoldPackageName(project.ModulePath),
			ImportPath:  project.ModulePath,
		}
	case 1:
		pkg := packages[0]
		if !token.IsIdentifier(pkg.Name) {
			return nil, fmt.Errorf("cannot scaffold Ferret composition for invalid package name %q", pkg.Name)
		}

		if err := ensureNewFerretAvailable(pkg); err != nil {
			return nil, err
		}

		scaffold = compositionScaffold{
			Directory:   pkg.Dir,
			PackageName: pkg.Name,
			ImportPath:  pkg.ImportPath,
		}
	default:
		candidates := make([]string, 0, len(packages))

		for _, pkg := range packages {
			candidates = append(candidates, pkg.ImportPath)
		}

		sort.Strings(candidates)

		return nil, fmt.Errorf(
			"no active ferret.New(...) composition found and the target package is ambiguous: %s; add one composition manually",
			strings.Join(candidates, ", "),
		)
	}

	scaffold.Filename = filepath.Join(scaffold.Directory, "ferret.go")

	if _, err := os.Stat(scaffold.Filename); err == nil {
		return nil, fmt.Errorf("cannot scaffold Ferret composition: %s already exists", scaffold.Filename)
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("stat composition scaffold %s: %w", scaffold.Filename, err)
	}

	return &scaffold, nil
}

func newScaffoldedComposition(scaffold *compositionScaffold) (*composition, error) {
	source := []byte(fmt.Sprintf(`package %s

import "github.com/MontFerret/ferret/v2"

// NewFerret creates a Ferret engine with the supplied options.
func NewFerret(options ...ferret.Option) (*ferret.Engine, error) {
	return ferret.New(options...)
}
`, scaffold.PackageName))
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, scaffold.Filename, source, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse generated Ferret composition: %w", err)
	}

	var call *ast.CallExpr

	ast.Inspect(file, func(node ast.Node) bool {
		candidate, ok := node.(*ast.CallExpr)
		if ok && isSelectorCall(candidate, "ferret", "New") {
			call = candidate
			return false
		}

		return true
	})

	if call == nil {
		return nil, fmt.Errorf("generated Ferret composition does not contain ferret.New(...)")
	}

	return &composition{
		Filename:  scaffold.Filename,
		Directory: scaffold.Directory,
		Package:   scaffold.ImportPath,
		Source:    source,
		Mode:      0o644,
		File:      file,
		FileSet:   fileSet,
		Call:      call,
		CoreAlias: "ferret",
	}, nil
}

func ensureProjectHasNoGoFiles(root string) error {
	return filepath.WalkDir(root, func(filename string, entry fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("inspect project for Go files: %w", err)
		}

		if entry.IsDir() {
			if filename != root && (entry.Name() == ".git" || entry.Name() == "vendor") {
				return filepath.SkipDir
			}

			return nil
		}

		if strings.EqualFold(filepath.Ext(entry.Name()), ".go") {
			return fmt.Errorf("cannot derive a composition package because %s is not part of a discoverable Go package", filename)
		}

		return nil
	})
}

func ensureNewFerretAvailable(pkg goPackageInfo) error {
	files := append(append(append([]string(nil), pkg.GoFiles...), pkg.CgoFiles...), pkg.TestGoFiles...)
	for _, name := range files {
		filename := filepath.Join(pkg.Dir, name)
		file, err := parser.ParseFile(token.NewFileSet(), filename, nil, 0)
		if err != nil {
			return fmt.Errorf("inspect %s for NewFerret: %w", filename, err)
		}

		if file.Name.Name != pkg.Name {
			continue
		}

		if file.Scope.Lookup("NewFerret") != nil {
			return fmt.Errorf("cannot scaffold Ferret composition: NewFerret is already declared in %s", filename)
		}
	}

	return nil
}

func scaffoldPackageName(modulePath string) string {
	leaf := path.Base(strings.TrimSpace(modulePath))
	var builder strings.Builder

	for index, character := range leaf {
		switch {
		case unicode.IsLetter(character) || character == '_' || (index > 0 && unicode.IsDigit(character)):
			builder.WriteRune(character)
		case index == 0 && unicode.IsDigit(character):
			builder.WriteString("project_")
			builder.WriteRune(character)
		default:
			builder.WriteRune('_')
		}
	}

	name := builder.String()
	if name == "" {
		name = "ferretproject"
	}

	if token.Lookup(name).IsKeyword() {
		name += "project"
	}

	if !token.IsIdentifier(name) {
		return "ferretproject"
	}

	return name
}
