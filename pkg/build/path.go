package build

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const artifactFileExtension = ".fqlc"

func artifactFileName(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)

	if ext == "" {
		return base + artifactFileExtension
	}

	return strings.TrimSuffix(base, ext) + artifactFileExtension
}

func siblingArtifactPath(path string) string {
	return filepath.Join(filepath.Dir(path), artifactFileName(path))
}

func canonicalPath(path string) (string, error) {
	resolved, err := filepath.Abs(path)

	if err != nil {
		return "", fmt.Errorf("resolve path %s: %w", path, err)
	}

	return filepath.Clean(resolved), nil
}

func samePath(left, right string) (bool, error) {
	leftPath, err := canonicalPath(left)

	if err != nil {
		return false, err
	}

	rightPath, err := canonicalPath(right)

	if err != nil {
		return false, err
	}

	if leftPath == rightPath {
		return true, nil
	}

	leftInfo, err := os.Stat(leftPath)
	if err != nil {
		return false, fmt.Errorf("inspect %s: %w", left, err)
	}

	rightInfo, err := os.Stat(rightPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}

		return false, fmt.Errorf("inspect %s: %w", right, err)
	}

	return os.SameFile(leftInfo, rightInfo), nil
}
