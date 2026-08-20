package migration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/MontFerret/ferret/v2/pkg/parser/fql"
	"github.com/MontFerret/ferret/v2/pkg/source"
)

const (
	finalForMessage  = "Final collecting FOR no longer becomes the script result in Ferret v2."
	finalForHelp     = "Add `return` before this loop."
	checkFailureHelp = "Fix this FQL source and rerun the check."
)

func checkFQLCompatibility(ctx context.Context, options CompatibilityOptions) (*CompatibilityResult, error) {
	from := options.From
	if from == "" {
		from = CompatibilityVersionV1
	}

	if from != CompatibilityVersionV1 {
		return nil, fmt.Errorf("unsupported compatibility source version %q: expected v1", from)
	}

	target := options.Path
	if target == "" {
		target = defaultDirectory
	}

	files, err := listCompatibilityFQLFiles(ctx, target)
	if err != nil {
		return nil, err
	}

	workingDirectory, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("resolve compatibility check working directory: %w", err)
	}

	result := &CompatibilityResult{ScannedFiles: len(files)}
	for _, filename := range files {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		data, err := os.ReadFile(filename)
		if err != nil {
			return nil, fmt.Errorf("read FQL source %s: %w", filename, err)
		}

		displayPath, err := compatibilityDisplayPath(workingDirectory, filename)
		if err != nil {
			return nil, err
		}

		src := source.New(displayPath, string(data))
		loop, err := finalTopLevelFQLFor(src)
		if err != nil {
			detail, line, column := fqlDiagnosticDetails(src, err)
			result.Diagnostics = append(result.Diagnostics, CompatibilityDiagnostic{
				Path:    displayPath,
				Message: "Could not check v1 compatibility: " + detail,
				Help:    checkFailureHelp,
				Line:    line,
				Column:  column,
				Kind:    CompatibilityDiagnosticFailure,
			})

			continue
		}

		if loop == nil {
			continue
		}

		line, column, err := fqlForLocation(src, loop)
		if err != nil {
			return nil, fmt.Errorf("locate compatibility issue in %s: %w", displayPath, err)
		}

		result.Diagnostics = append(result.Diagnostics, CompatibilityDiagnostic{
			Path:    displayPath,
			Message: finalForMessage,
			Help:    finalForHelp,
			Line:    line,
			Column:  column,
			Kind:    CompatibilityDiagnosticIssue,
		})
	}

	sort.Slice(result.Diagnostics, func(i, j int) bool {
		left := result.Diagnostics[i]
		right := result.Diagnostics[j]

		if left.Path != right.Path {
			return left.Path < right.Path
		}

		if left.Line != right.Line {
			return left.Line < right.Line
		}

		if left.Column != right.Column {
			return left.Column < right.Column
		}

		return left.Message < right.Message
	})

	return result, nil
}

func listCompatibilityFQLFiles(ctx context.Context, target string) ([]string, error) {
	target = filepath.Clean(target)

	info, err := os.Stat(target)
	if err != nil {
		return nil, fmt.Errorf("inspect compatibility check path %s: %w", target, err)
	}

	if !info.IsDir() {
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("compatibility check path is not a regular file or directory: %s", target)
		}

		if filepath.Ext(target) != ".fql" {
			return nil, fmt.Errorf("compatibility check file must use the .fql extension: %s", target)
		}

		return []string{filepath.Clean(target)}, nil
	}

	var files []string
	err = filepath.WalkDir(target, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if err := ctx.Err(); err != nil {
			return err
		}

		if entry.IsDir() {
			if path == target {
				return nil
			}

			if compatibilityDirectoryExcluded(entry.Name()) {
				return filepath.SkipDir
			}

			return nil
		}

		if filepath.Ext(entry.Name()) != ".fql" {
			return nil
		}

		info, err := compatibilityFileInfo(path, entry)
		if err != nil {
			return err
		}

		if info.Mode().IsRegular() {
			files = append(files, filepath.Clean(path))
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan FQL compatibility sources under %s: %w", target, err)
	}

	sort.Strings(files)

	return files, nil
}

func compatibilityDirectoryExcluded(name string) bool {
	switch name {
	case ".git", ".hg", ".svn", "node_modules", "vendor":
		return true
	default:
		return false
	}
}

func compatibilityFileInfo(path string, entry os.DirEntry) (os.FileInfo, error) {
	if entry.Type()&os.ModeSymlink != 0 {
		return os.Stat(path)
	}

	return entry.Info()
}

func compatibilityDisplayPath(workingDirectory, filename string) (string, error) {
	absolute := filename
	if !filepath.IsAbs(absolute) {
		absolute = filepath.Join(workingDirectory, absolute)
	}

	relative, err := filepath.Rel(workingDirectory, filepath.Clean(absolute))
	if err != nil {
		return "", fmt.Errorf("resolve invocation-relative FQL source path %s: %w", filename, err)
	}

	return filepath.ToSlash(relative), nil
}

func fqlForLocation(src *source.Source, loop fql.IForExpressionContext) (int, int, error) {
	token := loop.GetStart()
	if token == nil {
		return 0, 0, fmt.Errorf("final top-level FOR has no source token")
	}

	start, ok := fqlByteOffset(src.Content(), token.GetStart())
	if !ok {
		return 0, 0, fmt.Errorf("resolve final top-level FOR start")
	}

	end, ok := fqlByteOffset(src.Content(), token.GetStop()+1)
	if !ok || end <= start {
		return 0, 0, fmt.Errorf("resolve final top-level FOR span")
	}

	line, column := src.LocationAt(source.Span{Start: start, End: end})
	if line == 0 || column == 0 {
		return 0, 0, fmt.Errorf("resolve final top-level FOR location")
	}

	return line, column, nil
}
