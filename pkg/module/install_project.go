package module

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func discoverInstallProject(ctx context.Context, runner GoRunner, directory string) (*projectInfo, error) {
	if strings.TrimSpace(directory) == "" {
		directory = "."
	}

	absDirectory, err := filepath.Abs(directory)
	if err != nil {
		return nil, fmt.Errorf("resolve project directory: %w", err)
	}

	goModOutput, err := runner.Run(ctx, absDirectory, "env", "GOMOD")
	if err != nil {
		return nil, fmt.Errorf("locate project go.mod: %w", err)
	}

	goModPath := strings.TrimSpace(string(goModOutput))
	if goModPath == "" || goModPath == os.DevNull {
		return nil, fmt.Errorf("current directory is not inside a Go module; create go.mod before installing Ferret modules")
	}

	goModPath, err = filepath.Abs(goModPath)
	if err != nil {
		return nil, fmt.Errorf("resolve go.mod path: %w", err)
	}

	root := filepath.Dir(goModPath)
	moduleOutput, err := runner.Run(ctx, root, "list", "-m", "-json")
	if err != nil {
		return nil, fmt.Errorf("inspect project module: %w", err)
	}

	var projectModule goModuleInfo
	if err := json.NewDecoder(bytes.NewReader(moduleOutput)).Decode(&projectModule); err != nil {
		return nil, fmt.Errorf("decode project module metadata: %w", err)
	}
	if projectModule.Path == "" {
		return nil, fmt.Errorf("project go.mod does not declare a module path")
	}

	ferretOutput, err := runner.Run(ctx, root, "list", "-m", "-json", ferretCoreModulePath)
	if err != nil {
		return nil, fmt.Errorf("project does not select %s; add Ferret v2 before installing modules: %w", ferretCoreModulePath, err)
	}

	var ferretModule goModuleInfo
	if err := json.Unmarshal(ferretOutput, &ferretModule); err != nil {
		return nil, fmt.Errorf("decode project Ferret module metadata: %w", err)
	}
	if ferretModule.Path != ferretCoreModulePath || ferretModule.Version == "" {
		return nil, fmt.Errorf("project does not select a released %s version", ferretCoreModulePath)
	}
	if _, err := parseProjectFerretVersion(ferretModule.Version); err != nil {
		return nil, err
	}

	return &projectInfo{
		Root:          root,
		ModulePath:    projectModule.Path,
		GoModPath:     goModPath,
		GoSumPath:     filepath.Join(root, "go.sum"),
		FerretVersion: ferretModule.Version,
	}, nil
}

func discoverComposition(ctx context.Context, runner GoRunner, project *projectInfo) (*composition, error) {
	output, err := runner.Run(ctx, project.Root, "list", "-e", "-json", "./...")
	if err != nil {
		return nil, fmt.Errorf("enumerate project packages: %w", err)
	}

	decoder := json.NewDecoder(bytes.NewReader(output))
	var matches []*composition
	var dotImport string

	for {
		var pkg goPackageInfo
		if err := decoder.Decode(&pkg); err != nil {
			if err == io.EOF {
				break
			}

			return nil, fmt.Errorf("decode project package metadata: %w", err)
		}

		if pkg.Module == nil || !pkg.Module.Main {
			continue
		}

		files := append(append([]string(nil), pkg.GoFiles...), pkg.CgoFiles...)
		for _, name := range files {
			filename := filepath.Join(pkg.Dir, name)
			candidate, hasDotImport, err := inspectCompositionFile(filename, pkg.ImportPath)
			if err != nil {
				return nil, err
			}
			if hasDotImport && dotImport == "" {
				dotImport = filename
			}
			matches = append(matches, candidate...)
		}
	}

	if dotImport != "" {
		return nil, fmt.Errorf("dot-imported Ferret package in %s; use a normal or named import before installing modules", dotImport)
	}

	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("no active ferret.New(...) composition found in %s", project.Root)
	case 1:
		return matches[0], nil
	default:
		locations := make([]string, len(matches))
		for index, match := range matches {
			position := match.FileSet.Position(match.Call.Pos())
			locations[index] = fmt.Sprintf("%s:%d", position.Filename, position.Line)
		}

		return nil, fmt.Errorf("multiple active ferret.New(...) compositions found: %s", strings.Join(locations, ", "))
	}
}

func inspectCompositionFile(filename, importPath string) ([]*composition, bool, error) {
	source, err := os.ReadFile(filename)
	if err != nil {
		return nil, false, fmt.Errorf("read %s: %w", filename, err)
	}

	info, err := os.Stat(filename)
	if err != nil {
		return nil, false, fmt.Errorf("stat %s: %w", filename, err)
	}

	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, filename, source, parser.ParseComments)
	if err != nil {
		return nil, false, fmt.Errorf("parse %s: %w", filename, err)
	}

	coreAlias := ""
	dotImport := false
	for _, spec := range file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil || path != ferretCoreModulePath {
			continue
		}

		if spec.Name != nil {
			switch spec.Name.Name {
			case ".":
				dotImport = true
			case "_":
				continue
			default:
				coreAlias = spec.Name.Name
			}
		} else {
			coreAlias = "ferret"
		}
	}

	if coreAlias == "" {
		return nil, dotImport, nil
	}

	var matches []*composition
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok || !isSelectorCall(call, coreAlias, "New") {
			return true
		}

		matches = append(matches, &composition{
			Filename:  filename,
			Directory: filepath.Dir(filename),
			Package:   importPath,
			Source:    source,
			Mode:      info.Mode(),
			File:      file,
			FileSet:   fileSet,
			Call:      call,
			CoreAlias: coreAlias,
		})

		return true
	})

	return matches, dotImport, nil
}

func isSelectorCall(call *ast.CallExpr, qualifier, name string) bool {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != name {
		return false
	}

	ident, ok := selector.X.(*ast.Ident)
	return ok && ident.Name == qualifier
}
