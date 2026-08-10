package migration

import (
	"bytes"
	"context"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

var v1CompatibilityImports = map[string]string{
	"github.com/MontFerret/ferret":                          "github.com/MontFerret/ferret/v2/compat",
	"github.com/MontFerret/ferret/pkg/compiler":             "github.com/MontFerret/ferret/v2/compat/compiler",
	"github.com/MontFerret/ferret/pkg/runtime":              "github.com/MontFerret/ferret/v2/compat/runtime",
	"github.com/MontFerret/ferret/pkg/runtime/core":         "github.com/MontFerret/ferret/v2/compat/runtime/core",
	"github.com/MontFerret/ferret/pkg/runtime/values":       "github.com/MontFerret/ferret/v2/compat/runtime/values",
	"github.com/MontFerret/ferret/pkg/runtime/values/types": "github.com/MontFerret/ferret/v2/compat/runtime/values/types",
}

func planSourceChanges(ctx context.Context, project *migrationProject) (*sourcePlan, error) {
	result := &sourcePlan{ScannedFiles: len(project.GoFiles)}

	for _, filename := range project.GoFiles {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		snapshot, err := snapshotMigrationFile(filename)
		if err != nil {
			return nil, err
		}

		generated, err := migrationFileGenerated(filename, snapshot.Data)
		if err != nil {
			return nil, err
		}

		fileSet := token.NewFileSet()
		parseMode := parser.ParseComments
		if generated {
			parseMode |= parser.ImportsOnly
		}

		file, err := parser.ParseFile(fileSet, filename, snapshot.Data, parseMode)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", filename, err)
		}

		relative, err := filepath.Rel(project.Root, filename)
		if err != nil {
			return nil, fmt.Errorf("resolve project-relative path for %s: %w", filename, err)
		}
		if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("refusing to migrate Go file outside project root: %s", filename)
		}
		relative = filepath.ToSlash(relative)

		updated := 0
		for _, spec := range file.Imports {
			importPath, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				return nil, fmt.Errorf("decode import path in %s: %w", filename, err)
			}

			if isV2CompatImport(importPath) {
				result.HasCompat = true
				continue
			}

			replacement, supported := v1CompatibilityImports[importPath]
			if !supported {
				if isV1FerretImport(importPath) {
					result.RemainingV1 = true
					result.ManualActions = append(result.ManualActions, ManualAction{
						Path:       relative,
						ImportPath: importPath,
						Reason:     "no documented v2 compatibility replacement",
						Line:       fileSet.Position(spec.Pos()).Line,
					})
				}

				continue
			}

			if generated {
				result.RemainingV1 = true
				result.ManualActions = append(result.ManualActions, ManualAction{
					Path:       relative,
					ImportPath: importPath,
					Reason:     "generated file was not modified",
					Line:       fileSet.Position(spec.Pos()).Line,
				})

				continue
			}

			result.HasCompat = true
			spec.Path.Value = strconv.Quote(replacement)
			if importPath == v1ModulePath && spec.Name == nil {
				spec.Name = ast.NewIdent("ferret")
			}

			updated++
		}

		if updated == 0 {
			continue
		}

		var formatted bytes.Buffer
		if err := format.Node(&formatted, fileSet, file); err != nil {
			return nil, fmt.Errorf("format migrated source %s: %w", filename, err)
		}

		result.Changes = append(result.Changes, plannedChange{
			change: Change{
				Path:         relative,
				Before:       snapshot.Data,
				After:        formatted.Bytes(),
				BeforeExists: true,
			},
			before: snapshot,
			mode:   snapshot.Mode,
		})
		result.UpdatedImports += updated
		result.FormattedFiles++
	}

	sort.Slice(result.ManualActions, func(i, j int) bool {
		if result.ManualActions[i].Path != result.ManualActions[j].Path {
			return result.ManualActions[i].Path < result.ManualActions[j].Path
		}

		if result.ManualActions[i].Line != result.ManualActions[j].Line {
			return result.ManualActions[i].Line < result.ManualActions[j].Line
		}

		return result.ManualActions[i].ImportPath < result.ManualActions[j].ImportPath
	})

	return result, nil
}

func migrationFileGenerated(filename string, source []byte) (bool, error) {
	file, err := parser.ParseFile(token.NewFileSet(), filename, source, parser.PackageClauseOnly|parser.ParseComments)
	if err != nil {
		return false, fmt.Errorf("parse package declaration in %s: %w", filename, err)
	}

	return ast.IsGenerated(file), nil
}

func isV2CompatImport(importPath string) bool {
	return importPath == v2CompatPath || strings.HasPrefix(importPath, v2CompatPath+"/")
}

func isV1FerretImport(importPath string) bool {
	if importPath == v1ModulePath {
		return true
	}

	return strings.HasPrefix(importPath, v1ModulePath+"/") &&
		importPath != v2ModulePath &&
		!strings.HasPrefix(importPath, v2ModulePath+"/")
}
