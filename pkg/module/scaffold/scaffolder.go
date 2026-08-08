package scaffold

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"
	gomodule "golang.org/x/mod/module"

	modulemanifest "github.com/MontFerret/specs/pkg/module"
)

// Scaffolder creates a new Ferret module project without downloading dependencies.
type Scaffolder struct {
	environment EnvironmentProvider
}

// New constructs a project scaffolder.
func New(environment EnvironmentProvider) *Scaffolder {
	if environment == nil {
		environment = CurrentEnvironment
	}

	return &Scaffolder{environment: environment}
}

// Create generates and validates a module project before installing it at the destination.
func (s *Scaffolder) Create(ctx context.Context, options Options) (*Result, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if options.Name == "" {
		return nil, fmt.Errorf("module name is required")
	}

	if options.GoModule == "" {
		return nil, fmt.Errorf("--go-module is required")
	}

	if err := gomodule.CheckPath(options.GoModule); err != nil {
		return nil, fmt.Errorf("invalid Go module path %q: %w", options.GoModule, err)
	}

	leaf, err := moduleLeaf(options.Name)
	if err != nil {
		return nil, err
	}

	packageName := packageIdentifier(leaf)
	namespace := options.Namespace
	if namespace == "" {
		namespace = namespaceIdentifier(leaf)
	}

	manifest := modulemanifest.Manifest{
		Schema:        modulemanifest.SchemaV1,
		Name:          options.Name,
		Namespace:     namespace,
		Version:       "0.1.0",
		Description:   fmt.Sprintf("TODO: describe the %s Ferret module.", options.Name),
		License:       "LicenseRef-TODO",
		Documentation: "https://example.invalid/TODO",
	}

	if err := modulemanifest.Validate(&manifest); err != nil {
		return nil, fmt.Errorf("invalid module scaffold metadata: %w", err)
	}

	environment, err := s.environment()
	if err != nil {
		return nil, fmt.Errorf("resolve scaffold versions: %w", err)
	}

	if err := validateScaffoldEnvironment(environment); err != nil {
		return nil, err
	}

	destination := options.Directory
	if destination == "" {
		destination = leaf
	}

	destination, err = filepath.Abs(destination)
	if err != nil {
		return nil, fmt.Errorf("resolve scaffold destination: %w", err)
	}

	if _, err := os.Lstat(destination); err == nil {
		return nil, fmt.Errorf("destination %q already exists", destination)
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("inspect destination %q: %w", destination, err)
	}

	parent := filepath.Dir(destination)

	if err := os.MkdirAll(parent, 0o755); err != nil {
		return nil, fmt.Errorf("create destination parent %q: %w", parent, err)
	}

	staging, err := os.MkdirTemp(parent, ".ferret-module-")
	if err != nil {
		return nil, fmt.Errorf("create scaffold staging directory: %w", err)
	}

	defer os.RemoveAll(staging)

	manifestData, err := yaml.Marshal(&manifest)
	if err != nil {
		return nil, fmt.Errorf("encode module manifest: %w", err)
	}

	if err := writeScaffold(staging, scaffoldFiles(options, environment, packageName, manifestData)); err != nil {
		return nil, err
	}

	if _, err := modulemanifest.LoadFile(filepath.Join(staging, modulemanifest.ManifestFilename)); err != nil {
		return nil, fmt.Errorf("validate generated module manifest: %w", err)
	}

	if _, err := os.Lstat(destination); err == nil {
		return nil, fmt.Errorf("destination %q already exists", destination)
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("inspect destination %q: %w", destination, err)
	}

	if err := os.Rename(staging, destination); err != nil {
		return nil, fmt.Errorf("install scaffold at %q: %w", destination, err)
	}

	return &Result{Directory: destination, Namespace: namespace}, nil
}
