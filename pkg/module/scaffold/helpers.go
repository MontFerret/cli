package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	gomodule "golang.org/x/mod/module"

	modulemanifest "github.com/MontFerret/specs/pkg/module"
)

var goVersionPattern = regexp.MustCompile(`^[1-9][0-9]*\.[0-9]+(?:\.[0-9]+)?$`)

type scaffoldFile struct {
	path string
	data []byte
}

func moduleLeaf(name string) (string, error) {
	parts := strings.Split(name, "/")
	if len(parts) != 2 || parts[1] == "" {
		return "", fmt.Errorf("invalid Ferret module name %q: expected owner/name", name)
	}

	return parts[1], nil
}

func packageIdentifier(leaf string) string {
	value := normalizeIdentifier(leaf)

	if isGoKeyword(value) {
		value += "module"
	}

	return value
}

func namespaceIdentifier(leaf string) string {
	return normalizeIdentifier(leaf)
}

func normalizeIdentifier(value string) string {
	var builder strings.Builder

	for i, character := range value {
		valid := unicode.IsLetter(character) || character == '_' || (i > 0 && unicode.IsDigit(character))

		if valid {
			builder.WriteRune(character)
		} else if i == 0 && unicode.IsDigit(character) {
			builder.WriteString("module_")
			builder.WriteRune(character)
		} else {
			builder.WriteRune('_')
		}
	}

	return builder.String()
}

func isGoKeyword(value string) bool {
	switch value {
	case "break", "default", "func", "interface", "select", "case", "defer", "go", "map", "struct", "chan", "else", "goto", "package", "switch", "const", "fallthrough", "if", "range", "type", "continue", "for", "import", "return", "var":
		return true
	default:
		return false
	}
}

func validateScaffoldEnvironment(environment Environment) error {
	if !goVersionPattern.MatchString(environment.GoVersion) {
		return fmt.Errorf("invalid scaffold Go version %q", environment.GoVersion)
	}

	if !strings.HasPrefix(environment.FerretVersion, "v2.") {
		return fmt.Errorf("invalid scaffold Ferret version %q", environment.FerretVersion)
	}

	return nil
}

func validateAndResolveOptions(options Options) (Options, error) {
	if options.Name == "" {
		return Options{}, fmt.Errorf("module name is required")
	}

	if options.GoModule == "" {
		return Options{}, fmt.Errorf("--go-module is required")
	}

	if err := gomodule.CheckPath(options.GoModule); err != nil {
		return Options{}, fmt.Errorf("invalid Go module path %q: %w", options.GoModule, err)
	}

	leaf, err := moduleLeaf(options.Name)
	if err != nil {
		return Options{}, err
	}

	if options.Directory == "" {
		options.Directory = leaf
	}

	if options.Namespace == "" {
		options.Namespace = namespaceIdentifier(leaf)
	}

	if err := modulemanifest.Validate(newManifest(options)); err != nil {
		return Options{}, fmt.Errorf("invalid module scaffold metadata: %w", err)
	}

	return options, nil
}

func newManifest(options Options) *modulemanifest.Manifest {
	return &modulemanifest.Manifest{
		Schema:        modulemanifest.SchemaV1,
		Name:          options.Name,
		Namespace:     options.Namespace,
		Version:       "0.1.0",
		Description:   fmt.Sprintf("TODO: describe the %s Ferret module.", options.Name),
		License:       "LicenseRef-TODO",
		Documentation: "https://example.invalid/TODO",
	}
}

func scaffoldFiles(options Options, environment Environment, packageName string, manifest []byte) []scaffoldFile {
	return []scaffoldFile{
		{path: "ferret.yaml", data: manifest},
		{path: "go.mod", data: []byte(fmt.Sprintf("module %s\n\ngo %s\n\nrequire github.com/MontFerret/ferret/v2 %s\n", options.GoModule, environment.GoVersion, environment.FerretVersion))},
		{path: "module.go", data: []byte(fmt.Sprintf(`package %s

import (
	"github.com/MontFerret/ferret/v2/pkg/module"
	"github.com/MontFerret/ferret/v2/pkg/sdk"
)

// New returns the %s Ferret module.
func New() module.Module {
	return sdk.NewModule(%q, func(module.Bootstrap) error {
		// TODO: register the module's Ferret-facing functions.
		return nil
	})
}
`, packageName, options.Name, options.Name))},
		{path: "core/doc.go", data: []byte("// Package core contains the module's implementation.\npackage core\n")},
		{path: "lib/doc.go", data: []byte("// Package lib contains the Ferret-facing function bindings.\npackage lib\n")},
		{path: "README.md", data: []byte(fmt.Sprintf("# %s\n\nTODO: document this Ferret module.\n", options.Name))},
	}
}

func writeScaffold(root string, files []scaffoldFile) error {
	for _, file := range files {
		path := filepath.Join(root, filepath.FromSlash(file.path))

		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return fmt.Errorf("create scaffold directory for %s: %w", file.path, err)
		}

		if err := os.WriteFile(path, file.data, 0o644); err != nil {
			return fmt.Errorf("write scaffold file %s: %w", file.path, err)
		}
	}

	return nil
}
