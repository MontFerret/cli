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
	target, info, err := inspectMigrationTarget(directory)
	if err != nil {
		return nil, err
	}

	if info.Mode().IsRegular() {
		if filepath.Ext(target) != ".fql" {
			return nil, fmt.Errorf("migration file must use the .fql extension: %s", directory)
		}

		return &migrationProject{
			Root:     filepath.Dir(target),
			FQLFiles: []string{target},
		}, nil
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("migration path is not a regular FQL file or directory: %s", directory)
	}

	goModPath, hasGoModule, err := findContainingMigrationGoMod(ctx, target)
	if err != nil {
		return nil, err
	}

	root := target
	if hasGoModule {
		root = filepath.Dir(goModPath)
	}

	files, err := scanMigrationFiles(ctx, root)
	if err != nil {
		return nil, err
	}

	if len(files.Go) == 0 {
		return &migrationProject{Root: root, FQLFiles: files.FQL}, nil
	}

	if !hasGoModule {
		return nil, fmt.Errorf(
			"%w; Go source was found under %s, so run ferret migrate run from inside a Go module",
			goproject.ErrNoModule,
			root,
		)
	}

	return discoverGoMigrationProject(ctx, runner, target, goModPath, files)
}

func inspectMigrationTarget(target string) (string, os.FileInfo, error) {
	if strings.TrimSpace(target) == "" {
		target = defaultDirectory
	}

	absolute, err := filepath.Abs(target)
	if err != nil {
		return "", nil, fmt.Errorf("resolve migration path %s: %w", target, err)
	}

	info, err := os.Lstat(absolute)
	if err != nil {
		return "", nil, fmt.Errorf("inspect migration path %s: %w", target, err)
	}

	if info.Mode()&os.ModeSymlink != 0 {
		return "", nil, fmt.Errorf("refusing to migrate symlink target %s", target)
	}

	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", nil, fmt.Errorf("resolve migration path %s: %w", target, err)
	}

	return filepath.Clean(resolved), info, nil
}

func findContainingMigrationGoMod(ctx context.Context, directory string) (string, bool, error) {
	current := directory
	for {
		if err := ctx.Err(); err != nil {
			return "", false, err
		}

		candidate := filepath.Join(current, "go.mod")
		info, err := os.Lstat(candidate)
		if err == nil {
			if !info.Mode().IsRegular() {
				return "", false, fmt.Errorf("project go.mod is not a regular file: %s", candidate)
			}

			return candidate, true, nil
		}

		if !errors.Is(err, os.ErrNotExist) {
			return "", false, fmt.Errorf("inspect project go.mod %s: %w", candidate, err)
		}

		parent := filepath.Dir(current)
		if parent == current {
			return "", false, nil
		}

		current = parent
	}
}

func discoverGoMigrationProject(
	ctx context.Context,
	runner Runner,
	directory string,
	goModPath string,
	files migrationFiles,
) (*migrationProject, error) {
	discovered, err := goproject.Discover(ctx, runner, directory)
	if err != nil {
		if errors.Is(err, goproject.ErrNoModule) {
			return nil, fmt.Errorf("%w; run ferret migrate run from inside the project to migrate", err)
		}

		return nil, err
	}

	sameGoMod, err := sameMigrationFile(goModPath, discovered.GoModPath)
	if err != nil {
		return nil, err
	}

	if !sameGoMod {
		return nil, fmt.Errorf(
			"project root changed while discovering the project: found %s, Go reported %s",
			goModPath,
			discovered.GoModPath,
		)
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

	files, err = listMigrationGoFiles(ctx, runner, discovered.Root, files)
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

func sameMigrationFile(left, right string) (bool, error) {
	leftInfo, err := os.Stat(left)
	if err != nil {
		return false, fmt.Errorf("stat discovered project file %s: %w", left, err)
	}

	rightInfo, err := os.Stat(right)
	if err != nil {
		return false, fmt.Errorf("stat Go-reported project file %s: %w", right, err)
	}

	return os.SameFile(leftInfo, rightInfo), nil
}

func listMigrationFiles(ctx context.Context, runner Runner, root string) (migrationFiles, error) {
	files, err := scanMigrationFiles(ctx, root)
	if err != nil || len(files.Go) == 0 {
		return files, err
	}

	return listMigrationGoFiles(ctx, runner, root, files)
}

func scanMigrationFiles(ctx context.Context, root string) (migrationFiles, error) {
	goFiles := make(map[string]struct{})
	fqlFiles := make(map[string]struct{})
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

func listMigrationGoFiles(
	ctx context.Context,
	runner Runner,
	root string,
	files migrationFiles,
) (migrationFiles, error) {
	output, err := runner.Run(ctx, root, "list", "-e", "-json", "-mod=readonly", "./...")
	if err != nil {
		return migrationFiles{}, fmt.Errorf("enumerate project Go files: %w", err)
	}

	goFiles := make(map[string]struct{}, len(files.Go))
	for _, filename := range files.Go {
		goFiles[filename] = struct{}{}
	}

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

	result := migrationFiles{Go: make([]string, 0, len(goFiles)), FQL: files.FQL}

	for filename := range goFiles {
		result.Go = append(result.Go, filename)
	}

	sort.Strings(result.Go)

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
