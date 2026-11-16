package prompt

import (
	"context"
	"errors"
	"io"
	"os"
	"testing"
	"time"

	"github.com/chzyer/readline"
)

// Exercise the same process stream across two prompts. Keep the writer open
// during cancellation so EOF from the pipe cannot wake the first prompt.
func TestProcessStdinReusedAfterCancellation(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	defer reader.Close()
	defer writer.Close()
	previous := os.Stdin
	os.Stdin = reader
	defer func() { os.Stdin = previous }()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	seen := make(chan struct{})
	cfg := &readline.Config{FuncFilterInputRune: func(r rune) (rune, bool) {
		if r == 'h' {
			close(seen)
		}

		return r, true
	}}
	first, err := Open(ctx, cfg, reader, io.Discard, io.Discard)
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)
	go func() {
		_, err := first.Readline()
		done <- errors.Join(err, first.Close())
	}()

	if _, err := io.WriteString(writer, "h"); err != nil {
		t.Fatal(err)
	}

	select {
	case <-seen:
	case <-time.After(5 * time.Second):
		t.Fatal("first prompt never received input")
	}

	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("first prompt returned %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("process stdin did not wake")
	}

	second, err := Open(t.Context(), &readline.Config{}, reader, io.Discard, io.Discard)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := io.WriteString(writer, "next command\n"); err != nil {
		t.Fatal(err)
	}

	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	line, readErr := second.Readline()
	if err := errors.Join(readErr, second.Close()); err != nil || line != "next command" {
		t.Fatalf("second prompt got %q, error=%v", line, err)
	}
}
