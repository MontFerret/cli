package install

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func installPackageTarget(root, directory string) (string, error) {
	relative, err := filepath.Rel(root, directory)
	if err != nil {
		return "", fmt.Errorf("resolve composition package path: %w", err)
	}

	if relative == "." {
		return ".", nil
	}

	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("composition package %s is outside project root %s", directory, root)
	}

	return "./" + filepath.ToSlash(relative), nil
}

func relativeInstallPath(root, filename string) string {
	relative, err := filepath.Rel(root, filename)
	if err != nil {
		return filename
	}

	return filepath.ToSlash(relative)
}

func readOptionalInstallFile(path string) ([]byte, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}

		return nil, false, fmt.Errorf("read staged %s: %w", filepath.Base(path), err)
	}

	return data, true, nil
}

func installFileMode(snapshot fileSnapshot) os.FileMode {
	if snapshot.Exists {
		return snapshot.Mode
	}

	return 0o644
}
