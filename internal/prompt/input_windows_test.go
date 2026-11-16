package prompt

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"

	"github.com/chzyer/readline"
	"golang.org/x/sys/windows"
)

// Use a child console so the test never reads keystrokes from the caller.
func TestConsoleCancellation(t *testing.T) {
	const child = "FERRET_TEST_PRIVATE_CONSOLE"
	if os.Getenv(child) == "" {
		ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestConsoleCancellation$")
		cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_NEW_CONSOLE, HideWindow: true}
		cmd.Env = append(os.Environ(), child+"=1")

		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("console child: %v\n%s", err, output)
		}

		return
	}

	console, err := os.OpenFile("CONIN$", os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}

	defer console.Close()
	os.Stdin, syscall.Stdin = console, syscall.Handle(console.Fd())
	mode := func() uint32 {
		t.Helper()
		var mode uint32
		if err := windows.GetConsoleMode(windows.Handle(console.Fd()), &mode); err != nil {
			t.Fatal(err)
		}

		return mode
	}

	before := mode()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	cfg := &readline.Config{}
	entered := make(chan struct{})
	cfg.SetListener(func([]rune, int, rune) ([]rune, int, bool) {
		close(entered)
		return nil, 0, false
	})

	p, err := Open(ctx, cfg, console, os.Stdout, os.Stderr)
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)
	go func() {
		_, err := p.Readline()
		done <- err
	}()

	<-entered
	if mode()&windows.ENABLE_LINE_INPUT != 0 {
		t.Fatal("readline did not put the console in raw mode")
	}

	select {
	case err := <-done:
		t.Fatalf("console read returned before cancellation: %v", err)
	case <-time.After(100 * time.Millisecond):
	}

	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}

	if err := p.Close(); err != nil {
		t.Fatal(err)
	}

	if after := mode(); after != before {
		t.Fatalf("console mode changed from %#x to %#x", before, after)
	}
}
