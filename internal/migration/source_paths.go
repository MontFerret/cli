package migration

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

func migrationRelativePath(root, filename, kind string) (string, error) {
	relative, err := filepath.Rel(root, filename)
	if err != nil {
		return "", fmt.Errorf("resolve project-relative path for %s: %w", filename, err)
	}

	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("refusing to migrate %s file outside project root: %s", kind, filename)
	}

	return filepath.ToSlash(relative), nil
}

func sortManualActions(actions []ManualAction) {
	sort.Slice(actions, func(i, j int) bool {
		if actions[i].Path != actions[j].Path {
			return actions[i].Path < actions[j].Path
		}

		if actions[i].Line != actions[j].Line {
			return actions[i].Line < actions[j].Line
		}

		return actions[i].Detail < actions[j].Detail
	})
}
