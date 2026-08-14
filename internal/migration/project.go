package migration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/mod/modfile"

	"github.com/MontFerret/cli/v2/internal/goproject"
)

func discoverMigrationProject(ctx context.Context, runner Runner, directory string) (*migrationProject, error) {
	discovered, err := goproject.Discover(ctx, runner, directory)
	if err != nil {
		if errors.Is(err, goproject.ErrNoModule) {
			return nil, fmt.Errorf("%w; run ferret migrate from inside the project to migrate", err)
		}

		return nil, err
	}

	goMod, err := snapshotMigrationFile(discovered.GoModPath)
	if err != nil {
		return nil, err
	}

	parsed, err := modfile.Parse(discovered.GoModPath, goMod.Data, nil)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", discovered.GoModPath, err)
	}

	if parsed.Module == nil || parsed.Module.Mod.Path == "" {
		return nil, fmt.Errorf("project go.mod does not declare a module path")
	}

	if parsed.Module.Mod.Path != discovered.ModulePath {
		return nil, fmt.Errorf(
			"project module path changed while discovering the project: go list reported %q, go.mod declares %q",
			discovered.ModulePath,
			parsed.Module.Mod.Path,
		)
	}

	goSum, err := snapshotMigrationFile(discovered.GoSumPath)
	if err != nil {
		return nil, err
	}

	files, err := listMigrationFiles(ctx, runner, discovered.Root)
	if err != nil {
		return nil, err
	}

	return &migrationProject{
		Root:       discovered.Root,
		ModulePath: discovered.ModulePath,
		GoModPath:  discovered.GoModPath,
		GoSumPath:  discovered.GoSumPath,
		GoModFile:  parsed,
		GoFiles:    files.Go,
		FQLFiles:   files.FQL,
		GoMod:      goMod,
		GoSum:      goSum,
	}, nil
}

func listMigrationFiles(ctx context.Context, runner Runner, root string) (migrationFiles, error) {
	output, err := runner.Run(ctx, root, "list", "-e", "-json", "-mod=readonly", "./...")
	if err != nil {
		return migrationFiles{}, fmt.Errorf("enumerate project Go files: %w", err)
	}

	goFiles := make(map[string]struct{})
	fqlFiles := make(map[string]struct{})
	decoder := json.NewDecoder(bytes.NewReader(output))

	for {
		var pkg goPackageInfo
		if err := decoder.Decode(&pkg); err != nil {
			if err == io.EOF {
				break
			}

			return migrationFiles{}, fmt.Errorf("decode project package metadata: %w", err)
		}

		packageFiles := append([]string{}, pkg.GoFiles...)
		packageFiles = append(packageFiles, pkg.CgoFiles...)
		packageFiles = append(packageFiles, pkg.IgnoredGoFiles...)
		packageFiles = append(packageFiles, pkg.InvalidGoFiles...)
		packageFiles = append(packageFiles, pkg.TestGoFiles...)
		packageFiles = append(packageFiles, pkg.XTestGoFiles...)

		if pkg.Error != nil && pkg.Error.Err != "" &&
			(pkg.Module == nil || pkg.Dir == "" || len(packageFiles) == 0) {
			return migrationFiles{}, fmt.Errorf("enumerate package %q: %s", pkg.ImportPath, pkg.Error.Err)
		}

		if pkg.Module == nil || !pkg.Module.Main || pkg.Dir == "" {
			continue
		}

		for _, name := range packageFiles {
			if name == "" {
				continue
			}

			goFiles[filepath.Join(pkg.Dir, name)] = struct{}{}
		}
	}

	if err := walkMigrationFiles(ctx, root, goFiles, fqlFiles); err != nil {
		return migrationFiles{}, err
	}

	result := migrationFiles{
		Go:  make([]string, 0, len(goFiles)),
		FQL: make([]string, 0, len(fqlFiles)),
	}

	for filename := range goFiles {
		result.Go = append(result.Go, filename)
	}

	for filename := range fqlFiles {
		result.FQL = append(result.FQL, filename)
	}

	sort.Strings(result.Go)
	sort.Strings(result.FQL)

	return result, nil
}

func walkMigrationFiles(
	ctx context.Context,
	root string,
	goFiles map[string]struct{},
	fqlFiles map[string]struct{},
) error {
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if err := ctx.Err(); err != nil {
			return err
		}

		if entry.IsDir() {
			if path == root {
				return nil
			}

			name := entry.Name()
			if name == "vendor" || name == "testdata" || name == "node_modules" ||
				strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") {
				return filepath.SkipDir
			}

			if _, err := os.Stat(filepath.Join(path, "go.mod")); err == nil {
				return filepath.SkipDir
			} else if !os.IsNotExist(err) {
				return fmt.Errorf("inspect nested module boundary %s: %w", path, err)
			}

			return nil
		}

		switch filepath.Ext(entry.Name()) {
		case ".go":
			goFiles[path] = struct{}{}
		case ".fql":
			fqlFiles[path] = struct{}{}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("scan project source files: %w", err)
	}

	return nil
}
