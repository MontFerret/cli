package install

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

var renameInstallFile = os.Rename

func snapshotFile(path string) (fileSnapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fileSnapshot{Path: path, Mode: 0o644}, nil
		}

		return fileSnapshot{}, fmt.Errorf("read %s: %w", path, err)
	}

	info, err := os.Stat(path)
	if err != nil {
		return fileSnapshot{}, fmt.Errorf("stat %s: %w", path, err)
	}

	return fileSnapshot{Path: path, Data: data, Mode: info.Mode(), Exists: true}, nil
}

func commitInstallChanges(changes []fileChange) error {
	for _, change := range changes {
		if err := verifyInstallSnapshot(change.Before); err != nil {
			return err
		}
	}

	changes = changedInstallFiles(changes)
	if len(changes) == 0 {
		return nil
	}

	type preparedFile struct {
		change fileChange
		temp   string
	}

	prepared := make([]preparedFile, 0, len(changes))
	for _, change := range changes {
		mode := change.Mode
		if mode == 0 {
			mode = 0o644
		}

		temp, err := prepareInstallFile(change.Before.Path, change.After, mode)
		if err != nil {
			for _, item := range prepared {
				_ = os.Remove(item.temp)
			}

			return err
		}

		prepared = append(prepared, preparedFile{change: change, temp: temp})
	}

	committed := make([]fileSnapshot, 0, len(prepared))
	for index, item := range prepared {
		if err := renameInstallFile(item.temp, item.change.Before.Path); err != nil {
			for _, remaining := range prepared[index:] {
				_ = os.Remove(remaining.temp)
			}

			rollbackErr := rollbackInstallFiles(committed)

			return errors.Join(fmt.Errorf("replace %s: %w", item.change.Before.Path, err), rollbackErr)
		}

		committed = append(committed, item.change.Before)
	}

	return nil
}

func changedInstallFiles(changes []fileChange) []fileChange {
	result := make([]fileChange, 0, len(changes))
	for _, change := range changes {
		if change.Before.Exists && bytes.Equal(change.Before.Data, change.After) {
			continue
		}

		result = append(result, change)
	}

	return result
}

func verifyInstallSnapshot(snapshot fileSnapshot) error {
	current, err := snapshotFile(snapshot.Path)
	if err != nil {
		return err
	}

	if current.Exists != snapshot.Exists || !bytes.Equal(current.Data, snapshot.Data) {
		return fmt.Errorf("%s changed while module installation was running; no project files were updated", snapshot.Path)
	}

	return nil
}

func prepareInstallFile(destination string, data []byte, mode fs.FileMode) (string, error) {
	directory := filepath.Dir(destination)
	temp, err := os.CreateTemp(directory, ".ferret-mod-install-*")
	if err != nil {
		return "", fmt.Errorf("create temporary file for %s: %w", destination, err)
	}

	tempName := temp.Name()
	cleanup := func() {
		_ = temp.Close()
		_ = os.Remove(tempName)
	}

	if err := temp.Chmod(mode.Perm()); err != nil {
		cleanup()
		return "", fmt.Errorf("set temporary mode for %s: %w", destination, err)
	}

	if _, err := temp.Write(data); err != nil {
		cleanup()
		return "", fmt.Errorf("write temporary file for %s: %w", destination, err)
	}

	if err := temp.Sync(); err != nil {
		cleanup()
		return "", fmt.Errorf("sync temporary file for %s: %w", destination, err)
	}

	if err := temp.Close(); err != nil {
		_ = os.Remove(tempName)
		return "", fmt.Errorf("close temporary file for %s: %w", destination, err)
	}

	return tempName, nil
}

func rollbackInstallFiles(snapshots []fileSnapshot) error {
	var rollbackErr error

	for index := len(snapshots) - 1; index >= 0; index-- {
		snapshot := snapshots[index]
		if !snapshot.Exists {
			if err := os.Remove(snapshot.Path); err != nil && !errors.Is(err, os.ErrNotExist) {
				rollbackErr = errors.Join(rollbackErr, fmt.Errorf("remove %s during rollback: %w", snapshot.Path, err))
			}

			continue
		}

		temp, err := prepareInstallFile(snapshot.Path, snapshot.Data, snapshot.Mode)
		if err != nil {
			rollbackErr = errors.Join(rollbackErr, err)

			continue
		}

		if err := renameInstallFile(temp, snapshot.Path); err != nil {
			_ = os.Remove(temp)
			rollbackErr = errors.Join(rollbackErr, fmt.Errorf("restore %s: %w", snapshot.Path, err))
		}
	}

	return rollbackErr
}
