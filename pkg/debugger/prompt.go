package debugger

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/chzyer/readline"

	apidebug "github.com/MontFerret/api/debugger"
	"github.com/MontFerret/api/source"
	"github.com/MontFerret/cli/v2/internal/diagnostics"
	"github.com/MontFerret/cli/v2/internal/prompt"
)

type replState int

const (
	replStateReady replState = iota
	replStatePaused
	replStateCompleted
	replStateTerminated
)

// Start debugs a source file interactively using the standard streams.
func Start(ctx context.Context, session Session, src source.Source) error {
	return StartWithIO(ctx, session, src, os.Stdin, os.Stdout, os.Stderr)
}

// StartWithIO borrows the command streams and closes the session on every exit.
// A blocking injected reader must implement ReadContext(context.Context, []byte)
// (int, error) or be woken by its owner before StartWithIO can return.
func StartWithIO(ctx context.Context, session Session, src source.Source, stdin io.Reader, stdout, stderr io.Writer) (err error) {
	var rl *prompt.Prompt
	defer func() { err = errors.Join(err, session.Close(), rl.Close()) }()

	renderer := NewRenderer(stdout, src)
	state, err := startSession(ctx, session, renderer)
	if err != nil {
		return err
	}

	rl, err = prompt.Open(ctx, &readline.Config{
		Prompt: "(fdb) ", InterruptPrompt: "^C", EOFPrompt: "\n",
	}, stdin, stdout, stderr)
	if err != nil {
		return err
	}

	renderer.out = rl.Stdout()

	return runCommands(ctx, session, src.Name, renderer, state, rl)
}

// Run consumes commands from input and closes the session on exit. A caller
// supplying a blocking reader is responsible for waking it on cancellation.
func Run(ctx context.Context, session Session, src source.Source, input LineReader, out io.Writer) (err error) {
	defer func() { err = errors.Join(err, session.Close()) }()

	renderer := NewRenderer(out, src)
	state, err := startSession(ctx, session, renderer)
	if err != nil {
		return err
	}

	return runCommands(ctx, session, src.Name, renderer, state, input)
}

func startSession(ctx context.Context, session Session, renderer *Renderer) (replState, error) {
	fmt.Fprintln(renderer.out, "Ferret debugger started.")
	event, err := session.Start(ctx)
	state := nextReplState(replStateReady, event)

	if err := presentEvent(ctx, renderer, event, err); err != nil {
		return state, err
	}

	fmt.Fprintln(renderer.out, `Type "help" for available commands.`)

	return state, nil
}

func runCommands(ctx context.Context, session Session, sourceName string, renderer *Renderer, state replState, input LineReader) error {
	out := renderer.out
	var repeatCommand Command

	for {
		line, readErr := input.Readline()
		if cause := context.Cause(ctx); cause != nil {
			return cause
		}

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
			if repeatCommand.Name == CommandEmpty {
				continue
			}

			command = repeatCommand
		} else if isRepeatableCommand(command.Name) {
			repeatCommand = command
		} else {
			repeatCommand = Command{}
		}

		if message := unavailableCommandMessage(state, command.Name); message != "" {
			fmt.Fprintln(out, message)
			continue
		}

		if command.Name == CommandQuit {
			fmt.Fprintln(out, "Debug session terminated.")
			return nil
		}

		event, err := executeCommand(ctx, session, sourceName, renderer, command)
		if err != nil {
			return err
		}

		state = nextReplState(state, event)
	}
}

func executeCommand(ctx context.Context, session Session, mainSourceName string, renderer *Renderer, command Command) (*apidebug.Event, error) {
	switch command.Name {
	case CommandHelp:
		renderer.Help()
	case CommandBreak:
		location := command.Location

		if location.SourceName == "" {
			location.SourceName = mainSourceName
		}

		breakpoint, err := session.SetBreakpointAt(ctx, location, command.BreakpointOptions)
		if err != nil {
			return nil, commandError(ctx, renderer, "Breakpoint error", err)
		}

		renderer.BreakpointSet(breakpoint)
	case CommandDelete:
		err := session.DeleteBreakpoint(ctx, command.BreakpointID)
		if diagnostics.BreakpointNotFound(err) {
			fmt.Fprintf(renderer.out, "Unknown breakpoint: %d\n", command.BreakpointID)
			break
		}

		if err != nil {
			return nil, commandError(ctx, renderer, "Delete breakpoint error", err)
		}

		fmt.Fprintf(renderer.out, "Breakpoint %d deleted.\n", command.BreakpointID)
	case CommandBreakpoints:
		breakpoints, err := session.Breakpoints(ctx)
		if err != nil {
			return nil, commandError(ctx, renderer, "Breakpoints error", err)
		}

		renderer.Breakpoints(breakpoints)
	case CommandContinue:
		event, err := session.Continue(ctx)
		return event, presentEvent(ctx, renderer, event, err)
	case CommandStep:
		event, err := session.StepIn(ctx)
		return event, presentEvent(ctx, renderer, event, err)
	case CommandNext:
		event, err := session.StepOver(ctx)
		return event, presentEvent(ctx, renderer, event, err)
	case CommandOut:
		event, err := session.StepOut(ctx)
		return event, presentEvent(ctx, renderer, event, err)
	case CommandPause:
		if err := session.Pause(ctx); err != nil {
			return nil, commandError(ctx, renderer, "Pause error", err)
		}

		fmt.Fprintln(renderer.out, "Pause requested.")
	case CommandWhere:
		frames, err := session.Frames(ctx)
		if err != nil {
			return nil, commandError(ctx, renderer, "Stack error", err)
		}

		renderer.Frames(frames)
	case CommandLocals:
		locals, err := session.Locals(ctx)
		if err != nil {
			return nil, commandError(ctx, renderer, "Locals error", err)
		}

		renderer.Locals(locals)
	case CommandPrint:
		value, err := session.Evaluate(ctx, command.Argument)
		if err != nil {
			return nil, commandError(ctx, renderer, "Evaluation error", err)
		}

		renderer.Evaluation(value)
	}

	return nil, nil
}

func commandError(ctx context.Context, renderer *Renderer, prefix string, err error) error {
	if ctx.Err() != nil || diagnostics.DebuggerUnavailable(err) {
		return executionError(ctx, err)
	}

	renderer.Error(prefix, err)

	return nil
}

// A stop and an operation failure are independent UAPI results. In particular,
// completion content belongs to the user even if materialization cleanup failed.
func presentEvent(ctx context.Context, renderer *Renderer, event *apidebug.Event, err error) error {
	err = executionError(ctx, err)
	if event == nil {
		if err == nil {
			return errors.New("debugger returned no event")
		}

		// Execution may already have advanced past the previous stop.
		return err
	}

	// A terminated event may carry the very failure returned by the operation.
	// Render its location/status now and let the command boundary print that error.
	if event.Error != nil && err != nil && (errors.Is(err, event.Error) || diagnostics.Format(err) == diagnostics.Format(event.Error)) {
		snapshot := *event
		snapshot.Error = nil
		renderer.Event(&snapshot)
	} else {
		renderer.Event(event)
	}

	if ctx.Err() != nil || diagnostics.DebuggerUnavailable(err) || event.Reason == apidebug.ReasonCompleted || event.Reason == apidebug.ReasonTerminated {
		return err
	}

	// A returned stop is still inspectable, even if the operation reported an error.
	renderer.Error("Debugger error", err)

	return nil
}

func executionError(ctx context.Context, err error) error {
	cause := context.Cause(ctx)
	if cause == nil || errors.Is(err, cause) {
		return err
	}

	return errors.Join(cause, err)
}

func nextReplState(current replState, event *apidebug.Event) replState {
	if event == nil {
		return current
	}

	switch event.Reason {
	case apidebug.ReasonCompleted:
		return replStateCompleted
	case apidebug.ReasonTerminated:
		return replStateTerminated
	default:
		return replStatePaused
	}
}

func isRepeatableCommand(name CommandName) bool {
	switch name {
	case CommandContinue, CommandStep, CommandNext, CommandOut:
		return true
	default:
		return false
	}
}

func unavailableCommandMessage(state replState, name CommandName) string {
	switch name {
	case CommandContinue, CommandStep, CommandNext, CommandOut, CommandPause, CommandWhere, CommandLocals, CommandPrint:
	default:
		return ""
	}

	switch state {
	case replStateCompleted:
		return "Program has completed. Start a new debug session to continue debugging."
	case replStateTerminated:
		return "Program has terminated. Start a new debug session to continue debugging."
	default:
		return ""
	}
}
