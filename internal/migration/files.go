package migration

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

var renameMigrationFile = os.Rename

func snapshotMigrationFile(path string) (fileSnapshot, error) {
	info, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fileSnapshot{Path: path, Mode: 0o644}, nil
		}

		return fileSnapshot{}, fmt.Errorf("stat %s: %w", path, err)
	}

	if !info.Mode().IsRegular() {
		return fileSnapshot{}, fmt.Errorf("refusing to migrate non-regular file %s", path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fileSnapshot{}, fmt.Errorf("read %s: %w", path, err)
	}

	return fileSnapshot{Path: path, Data: data, Mode: info.Mode(), Exists: true}, nil
}

func readOptionalMigrationFile(path string) ([]byte, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, nil
		}

		return nil, false, fmt.Errorf("read staged %s: %w", path, err)
	}

	return data, true, nil
}

func migrationFileMode(snapshot fileSnapshot) fs.FileMode {
	if snapshot.Exists {
		return snapshot.Mode
	}

	return 0o644
}

func commitMigrationChanges(changes []plannedChange) error {
	for _, change := range changes {
		if err := verifyMigrationSnapshot(change.before); err != nil {
			return err
		}
	}

	type preparedFile struct {
		change plannedChange
		temp   string
	}

	prepared := make([]preparedFile, 0, len(changes))
	for _, change := range changes {
		temp, err := prepareMigrationFile(change.before.Path, change.change.After, change.mode)
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
		if err := renameMigrationFile(item.temp, item.change.before.Path); err != nil {
			for _, remaining := range prepared[index:] {
				_ = os.Remove(remaining.temp)
			}

			rollbackErr := rollbackMigrationFiles(committed)

			return errors.Join(fmt.Errorf("replace %s: %w", item.change.before.Path, err), rollbackErr)
		}

		committed = append(committed, item.change.before)
	}

	return nil
}

func verifyMigrationSnapshot(snapshot fileSnapshot) error {
	current, err := snapshotMigrationFile(snapshot.Path)
	if err != nil {
		return err
	}

	if current.Exists != snapshot.Exists ||
		current.Mode != snapshot.Mode ||
		!bytes.Equal(current.Data, snapshot.Data) {
		return fmt.Errorf("%s changed while migration was running; no project files were updated", snapshot.Path)
	}

	return nil
}

func prepareMigrationFile(destination string, data []byte, mode fs.FileMode) (string, error) {
	temp, err := os.CreateTemp(filepath.Dir(destination), ".ferret-migrate-*")
	if err != nil {
		return "", fmt.Errorf("create temporary file for %s: %w", destination, err)
	}

	tempName := temp.Name()
	cleanup := func() {
		_ = temp.Close()
		_ = os.Remove(tempName)
	}

	chmodMode := mode.Perm() | mode&(fs.ModeSetuid|fs.ModeSetgid|fs.ModeSticky)
	if err := temp.Chmod(chmodMode); err != nil {
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

func rollbackMigrationFiles(snapshots []fileSnapshot) error {
	var rollbackErr error

	for index := len(snapshots) - 1; index >= 0; index-- {
		snapshot := snapshots[index]
		if !snapshot.Exists {
			if err := os.Remove(snapshot.Path); err != nil && !errors.Is(err, os.ErrNotExist) {
				rollbackErr = errors.Join(rollbackErr, fmt.Errorf("remove %s during rollback: %w", snapshot.Path, err))
			}

			continue
		}

		temp, err := prepareMigrationFile(snapshot.Path, snapshot.Data, snapshot.Mode)
		if err != nil {
			rollbackErr = errors.Join(rollbackErr, err)
			continue
		}

		if err := renameMigrationFile(temp, snapshot.Path); err != nil {
			_ = os.Remove(temp)
			rollbackErr = errors.Join(rollbackErr, fmt.Errorf("restore %s: %w", snapshot.Path, err))
		}
	}

	return rollbackErr
}
