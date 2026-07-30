package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
)

func TestStoreUnsetRemovesOnlyPersistedValue(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	homedir.Reset()
	t.Cleanup(homedir.Reset)

	store, err := NewStore("ferret", "test")
	if err != nil {
		t.Fatal(err)
	}

	if err := store.Set(ExecRuntime, "builtin"); err != nil {
		t.Fatal(err)
	}
	if err := store.Set(PolicyFSRoot, "./fixtures"); err != nil {
		t.Fatal(err)
	}

	configFile := filepath.Join(home, ".ferret", "config.yaml")
	contents, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatal(err)
	}
	contents = append(contents, []byte("custom-setting: keep\n")...)
	if err := os.WriteFile(configFile, contents, 0o644); err != nil {
		t.Fatal(err)
	}

	homedir.Reset()
	store, err = NewStore("ferret", "test")
	if err != nil {
		t.Fatal(err)
	}

	if err := store.Unset(PolicyFSRoot); err != nil {
		t.Fatal(err)
	}

	got, err := store.Get(PolicyFSRoot)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf("expected filesystem root to be removed, got %v", got)
	}

	got, err = store.Get(ExecRuntime)
	if err != nil {
		t.Fatal(err)
	}
	if got != "builtin" {
		t.Fatalf("expected runtime to remain persisted, got %v", got)
	}

	contents, err = os.ReadFile(configFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(contents), "custom-setting: keep") {
		t.Fatalf("expected unknown config entry to be preserved:\n%s", contents)
	}
	if strings.Contains(string(contents), PolicyFSRoot+":") {
		t.Fatalf("expected filesystem root to be removed:\n%s", contents)
	}

	if err := store.Unset(PolicyFSRoot); err != nil {
		t.Fatalf("expected repeated unset to be idempotent, got %v", err)
	}
}

func TestStoreUnsetDoesNotAffectEnvironment(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("FERRET_RUNTIME", "https://runtime.example")
	homedir.Reset()
	t.Cleanup(homedir.Reset)

	store, err := NewStore("ferret", "test")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Set(ExecRuntime, "builtin"); err != nil {
		t.Fatal(err)
	}

	if err := store.Unset(ExecRuntime); err != nil {
		t.Fatal(err)
	}

	got, err := store.Get(ExecRuntime)
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://runtime.example" {
		t.Fatalf("expected environment runtime after unset, got %v", got)
	}
}

func TestStoreUnsetDoesNotAffectBoundFlags(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	homedir.Reset()
	t.Cleanup(homedir.Reset)

	store, err := NewStore("ferret", "test")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Set(ExecRuntime, "builtin"); err != nil {
		t.Fatal(err)
	}

	command := &cobra.Command{Use: "config-test"}
	command.Flags().String(ExecRuntime, "", "")
	if err := command.Flags().Set(ExecRuntime, "https://runtime.example"); err != nil {
		t.Fatal(err)
	}
	store.BindFlags(command)

	if err := store.Unset(ExecRuntime); err != nil {
		t.Fatal(err)
	}

	got, err := store.Get(ExecRuntime)
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://runtime.example" {
		t.Fatalf("expected bound flag runtime after unset, got %v", got)
	}
}

func TestStoreUnsetRejectsUnsupportedKey(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	homedir.Reset()
	t.Cleanup(homedir.Reset)

	store, err := NewStore("ferret", "test")
	if err != nil {
		t.Fatal(err)
	}

	if err := store.Unset("not-a-config-key"); !errors.Is(err, ErrInvalidFlag) {
		t.Fatalf("expected invalid flag error, got %v", err)
	}
}
