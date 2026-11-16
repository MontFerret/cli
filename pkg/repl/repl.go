package repl

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/chzyer/readline"

	"github.com/MontFerret/ferret/v2/pkg/source"

	"github.com/MontFerret/cli/v2/internal/diagnostics"
	"github.com/MontFerret/cli/v2/internal/output"
	"github.com/MontFerret/cli/v2/internal/prompt"
	"github.com/MontFerret/cli/v2/pkg/runtime"
)

// Start opens an interactive source shell using the process's standard streams.
func Start(ctx context.Context, opts runtime.Options, params map[string]interface{}) error {
	return StartWithIO(ctx, opts, params, os.Stdin, os.Stdout, os.Stderr)
}

// StartWithIO borrows command streams and owns its readline and runtime resources.
// Parent cancellation interrupts input and execution; active execution settles
// before the runtime and physical transport are released. A blocking injected
// reader must implement ReadContext(context.Context, []byte) (int, error) or be
// woken by its owner before StartWithIO can return.
func StartWithIO(ctx context.Context, opts runtime.Options, params map[string]any, stdin io.Reader, stdout, stderr io.Writer) (err error) {
	resources, err := runtime.New(ctx, opts)
	if err != nil {
		return err
	}

	var rl *prompt.Prompt
	defer func() { err = errors.Join(err, resources.Close(), rl.Close()) }()

	version, err := resources.Runtime.Version(ctx)
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

	ctx, stop := resources.CommandContext(ctx, opts.Endpoint)
	defer stop()

	rl, err = prompt.Open(ctx, &readline.Config{
		Prompt: "> ", InterruptPrompt: "^C", EOFPrompt: "exit",
	}, stdin, stdout, stderr)
	if err != nil {
		return err
	}

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
		resultErr, failure := output.Write(ctx, stdout, out, true)
		runErr = errors.Join(runErr, resultErr)

		if runErr != nil {
			diagnostics.Print(stderr, runErr)

			if ctx.Err() != nil {
				return errors.Join(context.Cause(ctx), runErr)
			}

			if failure == nil && diagnostics.Recoverable(runErr) {
				continue
			}

			if runtime.IsWireType(opts.Type) {
				return fmt.Errorf("wire runtime %s is unavailable; check the runtime host: %w", opts.Endpoint, runErr)
			}

			return runErr
		}
	}
}
