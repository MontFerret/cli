package cmd

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"

	"github.com/MontFerret/cli/v2/pkg/browser"
	"github.com/MontFerret/cli/v2/pkg/config"
	cliruntime "github.com/MontFerret/cli/v2/pkg/runtime"
	"github.com/MontFerret/ferret/v2/pkg/source"
)

var fsPolicyFlagNames = []string{
	config.PolicyFSRoot,
	config.PolicyFSReadOnly,
}

func TestFSPolicyFlagsAppearOnExecutionCommands(t *testing.T) {
	store := new(config.Store)
	commands := []*cobra.Command{
		RunCommand(store),
		ReplCommand(store),
		DebugCommand(store),
	}

	for _, command := range commands {
		t.Run(command.Name(), func(t *testing.T) {
			var output bytes.Buffer
			command.SetOut(&output)
			command.SetErr(&output)

			if err := command.Help(); err != nil {
				t.Fatal(err)
			}

			for _, name := range fsPolicyFlagNames {
				if command.Flags().Lookup(name) == nil {
					t.Fatalf("expected --%s to be registered", name)
				}
				if !strings.Contains(output.String(), "--"+name) {
					t.Fatalf("expected help to list --%s", name)
				}
			}

			if command.Flags().Lookup("runtime-fs-root") != nil {
				t.Fatal("expected --runtime-fs-root to be removed")
			}
		})
	}
}

func TestFSPolicyFlagsDoNotAppearOnVersion(t *testing.T) {
	command := VersionCommand(new(config.Store))

	for _, name := range fsPolicyFlagNames {
		if command.Flags().Lookup(name) != nil {
			t.Fatalf("expected --%s to be absent", name)
		}
	}
}

func TestFSPolicyFlagDefaultsDoNotCreateExplicitPolicy(t *testing.T) {
	command := &cobra.Command{Use: "policy-test"}
	addFSPolicyFlags(command)

	policy, err := fsPolicyFromCommand(command)
	if err != nil {
		t.Fatal(err)
	}
	if policy != nil {
		t.Fatalf("expected no explicit filesystem policy, got %#v", policy)
	}
}

func TestFSPolicyFlagValuesReachRuntimeOptions(t *testing.T) {
	command := &cobra.Command{Use: "policy-test"}
	addFSPolicyFlags(command)
	if err := command.Flags().Parse([]string{"--policy-fs-root= ./fixtures ", "--policy-fs-read-only"}); err != nil {
		t.Fatal(err)
	}

	policy, err := fsPolicyFromCommand(command)
	if err != nil {
		t.Fatal(err)
	}
	if policy == nil || policy.Root != "./fixtures" || !policy.ReadOnly {
		t.Fatalf("unexpected filesystem policy: %#v", policy)
	}
}

func TestFSPolicyFlagsRejectExplicitBlankRoot(t *testing.T) {
	command := &cobra.Command{Use: "policy-test"}
	addFSPolicyFlags(command)
	if err := command.Flags().Parse([]string{"--policy-fs-root= \t "}); err != nil {
		t.Fatal(err)
	}

	_, err := fsPolicyFromCommand(command)
	if err == nil || err.Error() != "--policy-fs-root cannot be empty" {
		t.Fatalf("expected blank root error, got %v", err)
	}
}

func TestFSPolicyEnvironmentValuesReachBuiltinRuntime(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "fixture.txt"), []byte("fixture"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("FERRET_POLICY_FS_ROOT", root)
	t.Setenv("FERRET_POLICY_FS_READ_ONLY", "true")

	store := newFSPolicyTestStore(t, t.TempDir())
	command := RunCommand(store)
	store.BindFlags(command)

	opts, err := runtimeOptionsFromCommand(command, store)
	if err != nil {
		t.Fatal(err)
	}
	if opts.FSPolicy == nil || opts.FSPolicy.Root != root || !opts.FSPolicy.ReadOnly {
		t.Fatalf("unexpected filesystem policy: %#v", opts.FSPolicy)
	}

	out, err := cliruntime.Run(
		context.Background(),
		opts,
		source.NewAnonymous(`RETURN TO_STRING(IO::FS::READ("fixture.txt"))`),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
}

func TestFSPolicyConfigValuesReachRuntimeOptions(t *testing.T) {
	home := t.TempDir()
	root := t.TempDir()
	store := newFSPolicyTestStore(t, home)
	if err := store.Set(config.PolicyFSRoot, root); err != nil {
		t.Fatal(err)
	}
	if err := store.Set(config.PolicyFSReadOnly, "true"); err != nil {
		t.Fatal(err)
	}

	homedir.Reset()
	store = newFSPolicyTestStore(t, home)
	command := RunCommand(store)
	store.BindFlags(command)

	opts, err := runtimeOptionsFromCommand(command, store)
	if err != nil {
		t.Fatal(err)
	}
	if opts.FSPolicy == nil || opts.FSPolicy.Root != root || !opts.FSPolicy.ReadOnly {
		t.Fatalf("unexpected filesystem policy: %#v", opts.FSPolicy)
	}
}

func TestLegacyRuntimeFSRootFlagIsRejected(t *testing.T) {
	command := RunCommand(new(config.Store))

	err := command.Flags().Parse([]string{"--runtime-fs-root=."})
	if err == nil || !strings.Contains(err.Error(), "unknown flag: --runtime-fs-root") {
		t.Fatalf("expected legacy flag rejection, got %v", err)
	}
}

func TestLegacyRuntimeFSRootEnvironmentIsIgnored(t *testing.T) {
	t.Setenv("FERRET_RUNTIME_FS_ROOT", t.TempDir())

	store := newFSPolicyTestStore(t, t.TempDir())
	command := RunCommand(store)
	store.BindFlags(command)

	opts, err := runtimeOptionsFromCommand(command, store)
	if err != nil {
		t.Fatal(err)
	}
	if opts.FSPolicy != nil {
		t.Fatalf("expected legacy environment variable to be ignored, got %#v", opts.FSPolicy)
	}
}

func TestExecuteRun_FSPolicyRemoteRuntimeRejectedBeforeBrowserStartup(t *testing.T) {
	opts := cliruntime.NewDefaultOptions()
	opts.Type = "https://worker.example"
	opts.WithBrowser = true
	opts.BrowserAddress = "://invalid"
	opts.FSPolicy = &cliruntime.FileSystemPolicy{ReadOnly: true}

	err := executeRun(
		newTestCommand(),
		opts,
		browser.Options{},
		nil,
		"RETURN 1",
		nil,
	)
	if !errors.Is(err, cliruntime.ErrFSPolicyRequiresBuiltinRuntime) {
		t.Fatalf("expected builtin runtime policy error, got %v", err)
	}
}

func newFSPolicyTestStore(t *testing.T, home string) *config.Store {
	t.Helper()

	t.Setenv("HOME", home)
	homedir.Reset()
	t.Cleanup(homedir.Reset)

	store, err := config.NewStore("ferret", "test")
	if err != nil {
		t.Fatal(err)
	}

	return store
}
