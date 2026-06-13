package debugcli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/chzyer/readline"

	"github.com/MontFerret/ferret/v2"
	"github.com/MontFerret/ferret/v2/pkg/runtime"
	"github.com/MontFerret/ferret/v2/pkg/source"
)

func Start(ctx context.Context, session Session, src *source.Source) error {
	rl, err := readline.NewEx(&readline.Config{
		Prompt:          "(fdb) ",
		InterruptPrompt: "^C",
		EOFPrompt:       "\n",
		Stdin:           os.Stdin,
		Stdout:          os.Stdout,
		Stderr:          os.Stderr,
	})
	if err != nil {
		return errors.Join(err, session.Close())
	}
	defer rl.Close()

	return Run(ctx, session, src, rl, rl.Stdout())
}

func Run(ctx context.Context, session Session, src *source.Source, input LineReader, out io.Writer) (err error) {
	defer func() {
		err = errors.Join(err, session.Close())
	}()

	renderer := NewRenderer(out, src)

	fmt.Fprintln(out, "Ferret debugger started.")
	event, err := session.Start(ctx)
	if err != nil {
		return err
	}
	renderer.Event(event)
	fmt.Fprintln(out, `Type "help" for available commands.`)

	for {
		line, readErr := input.Readline()
		if errors.Is(readErr, readline.ErrInterrupt) {
			continue
		}
		if errors.Is(readErr, io.EOF) {
			fmt.Fprintln(out, "Debug session terminated.")
			return nil
		}
		if readErr != nil {
			return readErr
		}

		command, parseErr := ParseCommand(line)
		if parseErr != nil {
			fmt.Fprintln(out, parseErr)
			continue
		}
		if command.Name == CommandEmpty {
			continue
		}

		quit := executeCommand(ctx, session, src.Name(), renderer, command)
		if quit {
			fmt.Fprintln(out, "Debug session terminated.")
			return nil
		}
	}
}

func executeCommand(ctx context.Context, session Session, mainFile string, renderer *Renderer, command Command) bool {
	switch command.Name {
	case CommandHelp:
		renderer.Help()
	case CommandBreak:
		location := command.Location
		if location.File == "" {
			location.File = mainFile
		}
		breakpoint, err := session.SetBreakpointAt(location, command.BreakpointOptions)
		if err != nil {
			renderer.Error("Breakpoint error", err)
		} else {
			renderer.BreakpointSet(breakpoint)
		}
	case CommandDelete:
		if err := session.DeleteBreakpoint(command.BreakpointID); err != nil {
			if errors.Is(err, runtime.ErrNotFound) {
				fmt.Fprintf(renderer.out, "Unknown breakpoint: %d\n", command.BreakpointID)
			} else {
				renderer.Error("Delete breakpoint error", err)
			}
		} else {
			fmt.Fprintf(renderer.out, "Breakpoint %d deleted.\n", command.BreakpointID)
		}
	case CommandBreakpoints:
		renderer.Breakpoints(session.Breakpoints())
	case CommandContinue:
		event, err := session.Continue(ctx)
		renderResume(event, err, renderer)
	case CommandStep:
		event, err := session.Step(ctx)
		renderResume(event, err, renderer)
	case CommandNext:
		event, err := session.Next(ctx)
		renderResume(event, err, renderer)
	case CommandOut:
		event, err := session.Out(ctx)
		renderResume(event, err, renderer)
	case CommandPause:
		if err := session.Pause(); err != nil {
			renderer.Error("Pause error", err)
		} else {
			fmt.Fprintln(renderer.out, "Pause requested.")
		}
	case CommandWhere:
		frames, err := session.Frames()
		if err != nil {
			renderer.Error("Stack error", err)
		} else {
			renderer.Frames(frames)
		}
	case CommandLocals:
		locals, err := session.Locals()
		if err != nil {
			renderer.Error("Locals error", err)
		} else {
			renderer.Locals(locals)
		}
	case CommandPrint:
		value, err := session.Evaluate(ctx, command.Argument)
		if err != nil {
			renderer.Error("Evaluation error", err)
		} else {
			renderer.Evaluation(value)
		}
	case CommandQuit:
		return true
	}

	return false
}

func renderResume(event *ferret.DebugEvent, err error, renderer *Renderer) {
	if err != nil {
		renderer.Error("Debugger error", err)
		return
	}
	renderer.Event(event)
}
