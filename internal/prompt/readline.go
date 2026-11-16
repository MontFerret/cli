// Package prompt manages the terminal input shared by interactive CLI commands.
package prompt

import (
	"context"
	"errors"
	"io"
	"os"

	"github.com/chzyer/readline"
)

// Open borrows the streams and interrupts Readline when ctx ends. The caller
// must invoke the returned cleanup after its last read.
func Open(ctx context.Context, cfg *readline.Config, stdin io.Reader, stdout, stderr io.Writer) (*readline.Instance, func() error, error) {
	input := readline.NewCancelableStdin(stdin)
	cfg.Stdin, cfg.Stdout, cfg.Stderr = input, stdout, stderr

	if stdin != os.Stdin || stdout != os.Stdout {
		// Injected streams must not manipulate the process terminal.
		cfg.FuncIsTerminal = func() bool { return false }
		cfg.FuncMakeRaw = func() error { return nil }
		cfg.FuncExitRaw = func() error { return nil }
		cfg.FuncOnWidthChanged = func(func()) {}
	}

	rl, err := readline.NewEx(cfg)
	if err != nil {
		return nil, nil, errors.Join(err, input.Close())
	}

	// Wake a blocked read, but let the caller close the terminal after the
	// readline operation has returned and its goroutines have initialized.
	closed := make(chan struct{})
	var inputErr error
	stop := context.AfterFunc(ctx, func() {
		inputErr = input.Close()
		close(closed)
	})

	return rl, func() error {
		if stop() {
			inputErr = input.Close()
		} else {
			<-closed
		}

		return errors.Join(inputErr, rl.Close())
	}, nil
}
