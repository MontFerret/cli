// Package prompt manages the terminal input shared by interactive CLI commands.
package prompt

import (
	"context"
	"errors"
	"io"
	"os"
	"syscall"

	"github.com/chzyer/readline"
)

// Prompt borrows its input and owns readline until Close. Readline and Close
// are called serially by the command loop.
type Prompt struct {
	*readline.Instance
	ctx        context.Context
	cancel     context.CancelFunc
	closeInput func() error
	readDone   <-chan struct{}
}

// Open attaches readline to the command streams. Injected readers may implement
// ReadContext to observe cancellation. Other blocking readers need their owner
// to wake an outstanding read before Close can return.
func Open(parent context.Context, cfg *readline.Config, stdin io.Reader, stdout, stderr io.Writer) (*Prompt, error) {
	ctx, cancel := context.WithCancel(parent)
	read := stdin.Read
	closeInput := func() error { return nil }

	if stdin == os.Stdin {
		var err error
		read, closeInput, err = processInput(ctx, os.Stdin)
		if err != nil {
			cancel()
			return nil, err
		}
	} else if input, ok := stdin.(interface {
		ReadContext(context.Context, []byte) (int, error)
	}); ok {
		read = func(b []byte) (int, error) { return input.ReadContext(ctx, b) }
	}

	cfg.Stdin = io.NopCloser(readFunc(func(b []byte) (int, error) {
		if ctx.Err() != nil {
			return 0, io.EOF
		}

		n, err := read(b)
		if ctx.Err() != nil {
			return 0, io.EOF
		}

		return n, err
	}))
	cfg.Stdout, cfg.Stderr = stdout, stderr

	if stdin != os.Stdin || stdout != os.Stdout {
		// Injected streams must not manipulate the process terminal.
		cfg.FuncIsTerminal = func() bool { return false }
		cfg.FuncMakeRaw = func() error { return nil }
		cfg.FuncExitRaw = func() error { return nil }
		cfg.FuncOnWidthChanged = func(func()) {}
	}

	rl, err := readline.NewEx(cfg)
	if err != nil {
		cancel()
		return nil, errors.Join(err, closeInput())
	}

	return &Prompt{Instance: rl, ctx: ctx, cancel: cancel, closeInput: closeInput}, nil
}

// Readline lets a canceled command release its runtime before a borrowed read
// settles. Close joins that read before returning the streams to the caller.
func (p *Prompt) Readline() (string, error) {
	done := make(chan struct{})
	p.readDone = done
	var line string
	var err error
	go func() {
		defer close(done)
		line, err = p.Instance.Readline()
	}()

	select {
	case <-done:
		if cause := context.Cause(p.ctx); cause != nil {
			return "", cause
		}

		return line, err
	case <-p.ctx.Done():
		return "", context.Cause(p.ctx)
	}
}

// Close waits for readline to leave raw mode and releases prompt-owned handles.
func (p *Prompt) Close() error {
	if p == nil {
		return nil
	}

	p.cancel()
	if p.readDone != nil {
		<-p.readDone
	}

	err := p.Instance.Close()
	// readline returns Windows' successful SetConsoleMode result as errno 0.
	if err == syscall.Errno(0) {
		err = nil
	}

	return errors.Join(err, p.closeInput())
}
