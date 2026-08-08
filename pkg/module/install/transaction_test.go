package install

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommitInstallChangesRollsBackEarlierFiles(t *testing.T) {
	directory := t.TempDir()
	firstPath := filepath.Join(directory, "first.go")
	secondPath := filepath.Join(directory, "go.mod")
	writeInstallTestFile(t, firstPath, "first-before")
	writeInstallTestFile(t, secondPath, "second-before")

	first, err := snapshotFile(firstPath)
	if err != nil {
		t.Fatal(err)
	}
	second, err := snapshotFile(secondPath)
	if err != nil {
		t.Fatal(err)
	}

	originalRename := renameInstallFile
	renameCount := 0
	renameInstallFile = func(oldPath, newPath string) error {
		renameCount++
		if renameCount == 2 {
			return errors.New("injected rename failure")
		}
		return os.Rename(oldPath, newPath)
	}
	t.Cleanup(func() { renameInstallFile = originalRename })

	err = commitInstallChanges([]fileChange{
		{Before: first, After: []byte("first-after"), Mode: first.Mode},
		{Before: second, After: []byte("second-after"), Mode: second.Mode},
	})
	if err == nil || !strings.Contains(err.Error(), "injected rename failure") {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := readInstallTestFile(t, firstPath); got != "first-before" {
		t.Fatalf("first file was not rolled back: %q", got)
	}
	if got := readInstallTestFile(t, secondPath); got != "second-before" {
		t.Fatalf("second file changed: %q", got)
	}
}

func TestCommitInstallChangesPreservesModeAndDetectsConcurrentChange(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "main.go")
	if err := os.WriteFile(path, []byte("before"), 0o640); err != nil {
		t.Fatal(err)
	}
	snapshot, err := snapshotFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if err := commitInstallChanges([]fileChange{{Before: snapshot, After: []byte("after"), Mode: snapshot.Mode}}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o640 {
		t.Fatalf("mode changed to %o", info.Mode().Perm())
	}

	concurrent, err := snapshotFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("user edit"), 0o640); err != nil {
		t.Fatal(err)
	}
	err = commitInstallChanges([]fileChange{{Before: concurrent, After: []byte("installer edit"), Mode: concurrent.Mode}})
	if err == nil || !strings.Contains(err.Error(), "changed while module installation was running") {
		t.Fatalf("unexpected concurrent edit error: %v", err)
	}
	if got := readInstallTestFile(t, path); got != "user edit" {
		t.Fatalf("concurrent edit was overwritten: %q", got)
	}
}

func TestCommitInstallChangesDoesNotOverwriteConcurrentNewFile(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "ferret.go")
	snapshot, err := snapshotFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Exists {
		t.Fatal("new composition unexpectedly existed")
	}

	if err := os.WriteFile(path, []byte("user composition"), 0o644); err != nil {
		t.Fatal(err)
	}
	err = commitInstallChanges([]fileChange{{Before: snapshot, After: []byte("generated composition"), Mode: 0o644}})
	if err == nil || !strings.Contains(err.Error(), "changed while module installation was running") {
		t.Fatalf("unexpected concurrent creation error: %v", err)
	}
	if got := readInstallTestFile(t, path); got != "user composition" {
		t.Fatalf("concurrent composition was overwritten: %q", got)
	}
}
