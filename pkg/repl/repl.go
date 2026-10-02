package repl

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/chzyer/readline"

	"github.com/MontFerret/ferret/v2/pkg/source"

	"github.com/MontFerret/cli/v2/internal/diagnostics"
	"github.com/MontFerret/cli/v2/pkg/runtime"
)

// Start opens an interactive source shell using the process's standard streams.
func Start(ctx context.Context, opts runtime.Options, params map[string]interface{}) error {
	return StartWithIO(ctx, opts, params, os.Stdin, os.Stdout, os.Stderr)
}

// StartWithIO borrows command streams and owns its readline and runtime resources.
// Parent cancellation interrupts input and execution; active execution settles
// before the runtime and physical transport are released.
func StartWithIO(ctx context.Context, opts runtime.Options, params map[string]any, stdin io.Reader, stdout, stderr io.Writer) (err error) {
	resources, err := runtime.OpenSource(ctx, opts)
	if err != nil {
		return err
	}

	defer func() { err = errors.Join(err, resources.Close()) }()

	var version string
	if resources.Legacy != nil {
		version, err = resources.Legacy.Version(ctx)
	} else {
		metadata, versionErr := resources.Runtime.Version(ctx)
		version, err = metadata.String(), versionErr
	}

	if err != nil {
		return err
	}

	if runtime.IsWireType(opts.Type) {
		_, err = fmt.Fprintf(stdout, "Welcome to Ferret REPL (Wire runtime: %s; version: %s)\n", opts.Endpoint, version)
	} else {
		_, err = fmt.Fprintf(stdout, "Welcome to Ferret REPL %s\n", version)
	}

	if err != nil {
		return err
	}

	if _, err := fmt.Fprintln(stdout, "Please use `exit` or `Ctrl-D` to exit this program."); err != nil {
		return err
	}

	input := readline.NewCancelableStdin(stdin)
	defer input.Close()

	readlineConfig := &readline.Config{
		Prompt: "> ", InterruptPrompt: "^C", EOFPrompt: "exit",
		Stdin: input, Stdout: stdout, Stderr: stderr,
	}

	if stdin != os.Stdin || stdout != os.Stdout {
		// Injected streams must not manipulate the process terminal or depend
		// on whether the calling test or embedding application has a TTY.
		readlineConfig.FuncIsTerminal = func() bool { return false }
		readlineConfig.FuncMakeRaw = func() error { return nil }
		readlineConfig.FuncExitRaw = func() error { return nil }
		readlineConfig.FuncOnWidthChanged = func(func()) {}
	}

	rl, err := readline.NewEx(readlineConfig)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	if resources.Disconnected != nil {
		watched := make(chan struct{})
		go func() {
			defer close(watched)

			select {
			case <-ctx.Done():
			case <-resources.Disconnected:
				cancel(fmt.Errorf("wire runtime %s disconnected; check the runtime host", opts.Endpoint))
			}
		}()
		defer func() {
			cancel(nil)
			<-watched
		}()
	}

	var closeOnce sync.Once
	var closeErr error
	closeInput := func() {
		closeOnce.Do(func() {
			// Interrupt only input asynchronously. Close readline after Readline
			// returns so its terminal goroutine has completed initialization.
			closeErr = input.Close()
		})
	}
	closed := make(chan struct{})
	stop := context.AfterFunc(ctx, func() {
		closeInput()
		close(closed)
	})
	defer func() {
		if !stop() {
			<-closed
		}

		closeInput()
		err = errors.Join(err, closeErr, rl.Close())
	}()

	var commands []string
	var multiline bool

	for {
		line, readErr := rl.Readline()
		if ctx.Err() != nil {
			return context.Cause(ctx)
		}

		if readErr != nil {
			if errors.Is(readErr, io.EOF) || errors.Is(readErr, readline.ErrInterrupt) {
				return nil
			}

			return readErr
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "%") {
			line = line[1:]
			multiline = !multiline
		}

		if multiline {
			commands = append(commands, line)
			continue
		}

		commands = append(commands, line)
		query := strings.TrimSpace(strings.Join(commands, "\n"))
		commands = make([]string, 0, 10)
		if query == "" {
			continue
		}

		if query == "exit" {
			return nil
		}

		out, runErr := runtime.RunSource(ctx, resources, source.NewAnonymous(query), params)
		if out != nil {
			if outputErr := writeResult(stdout, out); outputErr != nil {
				return errors.Join(runErr, outputErr)
			}
		}

		if runErr != nil {
			diagnostics.Print(stderr, runErr)

			if ctx.Err() != nil {
				return errors.Join(context.Cause(ctx), runErr)
			}

			if diagnostics.Recoverable(runErr) || resources.Legacy != nil {
				continue
			}

			if runtime.IsWireType(opts.Type) {
				return fmt.Errorf("wire runtime %s is unavailable; check the runtime host: %w", opts.Endpoint, runErr)
			}

			return runErr
		}
	}
}
