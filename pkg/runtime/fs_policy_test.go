package runtime

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MontFerret/ferret/v2"
	"github.com/MontFerret/ferret/v2/pkg/source"
)

func TestBuiltinFilesystemDefaultsToWritableCurrentDirectory(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)

	out, err := Run(
		context.Background(),
		NewDefaultOptions(),
		source.NewAnonymous(`
IO::FS::WRITE("output.txt", TO_BINARY("written"))
RETURN true
`),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()

	content, err := os.ReadFile(filepath.Join(root, "output.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "written" {
		t.Fatalf("unexpected output content: %q", content)
	}
}

func TestBuiltinFilesystemPolicyUsesConfiguredRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "fixture.txt"), []byte("fixture"), 0o644); err != nil {
		t.Fatal(err)
	}

	opts := NewDefaultOptions()
	opts.FSPolicy = &FileSystemPolicy{Root: root}
	out, err := Run(
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

func TestBuiltinFilesystemPolicyRejectsRootEscape(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "root")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(parent, "outside.txt"), []byte("outside"), 0o644); err != nil {
		t.Fatal(err)
	}

	opts := NewDefaultOptions()
	opts.FSPolicy = &FileSystemPolicy{Root: root}
	_, err := Run(
		context.Background(),
		opts,
		source.NewAnonymous(`RETURN IO::FS::READ("../outside.txt")`),
		nil,
	)
	if err == nil {
		t.Fatal("expected root escape to fail")
	}
}

func TestBuiltinFilesystemPolicyEnforcesReadOnly(t *testing.T) {
	root := t.TempDir()
	opts := NewDefaultOptions()
	opts.FSPolicy = &FileSystemPolicy{Root: root, ReadOnly: true}

	_, err := Run(
		context.Background(),
		opts,
		source.NewAnonymous(`
IO::FS::WRITE("output.txt", TO_BINARY("blocked"))
RETURN true
`),
		nil,
	)
	if err == nil || !strings.Contains(err.Error(), "filesystem is read-only") {
		t.Fatalf("expected read-only error, got %v", err)
	}

	if _, statErr := os.Stat(filepath.Join(root, "output.txt")); !os.IsNotExist(statErr) {
		t.Fatalf("expected output file not to exist, got %v", statErr)
	}
}

func TestBuiltinFilesystemPolicyRejectsMissingRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "missing")
	opts := NewDefaultOptions()
	opts.FSPolicy = &FileSystemPolicy{Root: root}

	_, err := NewBuiltin(opts)
	if err == nil || !strings.Contains(err.Error(), root) {
		t.Fatalf("expected filesystem root error, got %v", err)
	}
}

func TestNewRejectsFilesystemPolicyForRemoteRuntime(t *testing.T) {
	opts := NewDefaultOptions()
	opts.Type = "https://worker.example"
	opts.FSPolicy = &FileSystemPolicy{ReadOnly: true}

	_, err := New(opts)
	if !errors.Is(err, ErrFSPolicyRequiresBuiltinRuntime) {
		t.Fatalf("expected builtin runtime policy error, got %v", err)
	}
}

func TestDebugSessionUsesConfiguredFilesystemPolicy(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "fixture.txt"), []byte("fixture"), 0o644); err != nil {
		t.Fatal(err)
	}

	opts := NewDefaultOptions()
	opts.FSPolicy = &FileSystemPolicy{Root: root, ReadOnly: true}
	session, err := NewDebugSession(
		context.Background(),
		opts,
		nil,
		source.NewAnonymous(`RETURN TO_STRING(IO::FS::READ("fixture.txt"))`),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	event, err := session.Start(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if event.Reason != ferret.DebugReasonEntry {
		t.Fatalf("expected debugger entry event, got %#v", event)
	}

	event, err = session.Continue(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if event.Reason != ferret.DebugReasonCompleted {
		t.Fatalf("expected debugger completion event, got %#v", event)
	}
}
