package config

import (
	"errors"
	"testing"

	"github.com/mitchellh/go-homedir"
)

func TestFilesystemPolicyConfigKeysReplaceRuntimeFSRoot(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	homedir.Reset()
	t.Cleanup(homedir.Reset)

	store, err := NewStore("ferret", "test")
	if err != nil {
		t.Fatal(err)
	}

	if err := store.Set(PolicyFSRoot, "./fixtures"); err != nil {
		t.Fatal(err)
	}
	if err := store.Set(PolicyFSReadOnly, "true"); err != nil {
		t.Fatal(err)
	}

	if _, err := store.Get(PolicyFSRoot); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(PolicyFSReadOnly); err != nil {
		t.Fatal(err)
	}
	if err := store.Set("runtime-fs-root", "./legacy"); !errors.Is(err, ErrInvalidFlag) {
		t.Fatalf("expected legacy config key rejection, got %v", err)
	}
	if _, err := store.Get("runtime-fs-root"); !errors.Is(err, ErrInvalidFlag) {
		t.Fatalf("expected legacy config key lookup rejection, got %v", err)
	}

	seen := make(map[string]bool)
	for _, entry := range store.List() {
		seen[entry.Key] = true
	}

	if !seen[PolicyFSRoot] || !seen[PolicyFSReadOnly] {
		t.Fatalf("expected filesystem policy keys in config list: %#v", seen)
	}
	if seen["runtime-fs-root"] {
		t.Fatal("expected runtime-fs-root to be removed from config list")
	}
}
