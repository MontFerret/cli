package debugger

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"text/tabwriter"

	apidebug "github.com/MontFerret/api/debugger"
	"github.com/MontFerret/api/source"
	"github.com/MontFerret/cli/v2/internal/diagnostics"
	native "github.com/MontFerret/ferret/v2/pkg/source"
)

const helpText = `Commands:
  break, b <location>           Set at next executable location in file
  break --exact <location>      Set only at the exact executable location
  break --next <location>       Set at next executable location in file
  break --in-function <location> Set at next executable location in function
  breakpoints, bp, bl           List breakpoints
  delete, d <id>                Delete breakpoint
  continue, c                   Resume execution
  step, s                       Step into next source location
  next, n                       Step over current source location
  out                           Step out of current frame
  pause                         Request pause after the next resume (Wire alpha.4 rejects this while stopped)
  where, w, bt                  Show stack trace
  locals, l                     Show local variables
  print, p, eval, e <expr>      Evaluate a safe expression (no calls, queries, or mutation)
  quit, q                       Stop debugging and exit

Locations: 12, 12:4, file.fql:12, file.fql:12:4`

// Renderer presents portable debugger responses against the original source file.
type Renderer struct {
	out    io.Writer
	source native.Source
}

// NewRenderer indexes the source once for snippets throughout this session.
func NewRenderer(out io.Writer, src source.Source) *Renderer {
	return &Renderer{out: out, source: native.New(src.Name, src.Content)}
}

func (r *Renderer) Help() {
	fmt.Fprintln(r.out, helpText)
}

func (r *Renderer) Event(event *apidebug.Event) {
	if event == nil {
		fmt.Fprintln(r.out, "Debugger returned no event.")
		return
	}

	switch event.Reason {
	case apidebug.ReasonEntry:
		fmt.Fprintf(r.out, "Paused at %s\n", formatLocation(event.Location.Location))
		r.snippet(event.Location)
	case apidebug.ReasonBreakpoint:
		switch len(event.HitBreakpointIDs) {
		case 0:
			fmt.Fprintf(r.out, "Paused on breakpoint at %s\n", formatLocation(event.Location.Location))
		case 1:
			fmt.Fprintf(r.out, "Paused on breakpoint %d at %s\n", event.HitBreakpointIDs[0], formatLocation(event.Location.Location))
		default:
			fmt.Fprintf(r.out, "Paused on breakpoints %s at %s\n", formatBreakpointIDs(event.HitBreakpointIDs), formatLocation(event.Location.Location))
		}

		r.snippet(event.Location)
	case apidebug.ReasonStep:
		fmt.Fprintf(r.out, "Paused after step at %s\n", formatLocation(event.Location.Location))
		r.snippet(event.Location)
	case apidebug.ReasonPause:
		fmt.Fprintf(r.out, "Paused on pause request at %s\n", formatLocation(event.Location.Location))
		r.snippet(event.Location)
	case apidebug.ReasonRuntimeError:
		fmt.Fprintln(r.out, "Paused on runtime error.")
	case apidebug.ReasonCompleted:
		fmt.Fprintln(r.out, "Program completed.")
	case apidebug.ReasonTerminated:
		fmt.Fprintln(r.out, "Program terminated.")
	default:
		fmt.Fprintf(r.out, "Debugger stopped: %s\n", event.Reason)
	}

	if event.Output != nil {
		fmt.Fprintln(r.out, "Result:")
		fmt.Fprintln(r.out, string(event.Output.Data))
	}

	diagnostics.Print(r.out, event.Error)
}

func (r *Renderer) BreakpointSet(breakpoint apidebug.Breakpoint) {
	requested := formatLocation(breakpoint.RequestedLocation)
	mode := formatBindingMode(breakpoint.BindingMode)

	if !breakpoint.Bound {
		fmt.Fprintf(r.out, "Breakpoint %d could not be bound at %s (%s).\n", breakpoint.ID, requested, mode)
		return
	}

	bound := formatLocation(breakpoint.Location.Location)
	if requested == bound {
		fmt.Fprintf(r.out, "Breakpoint %d set at %s (%s).\n", breakpoint.ID, bound, mode)
		return
	}

	fmt.Fprintf(r.out, "Breakpoint %d set at %s (requested %s, %s).\n", breakpoint.ID, bound, requested, mode)
}

func (r *Renderer) Breakpoints(breakpoints []apidebug.Breakpoint) {
	if len(breakpoints) == 0 {
		fmt.Fprintln(r.out, "No breakpoints.")
		return
	}

	table := tabwriter.NewWriter(r.out, 0, 4, 2, ' ', 0)
	fmt.Fprintln(table, "ID\tRequested\tBound\tMode\tState")

	for _, breakpoint := range breakpoints {
		requested := formatLocation(breakpoint.RequestedLocation)
		bound := "-"
		state := "unbound"

		if breakpoint.Bound {
			bound = formatLocation(breakpoint.Location.Location)
			state = "bound"
		}

		fmt.Fprintf(table, "%d\t%s\t%s\t%s\t%s\n", breakpoint.ID, requested, bound, formatBindingMode(breakpoint.BindingMode), state)
	}

	_ = table.Flush()
}

func (r *Renderer) Frames(frames []apidebug.Frame) {
	if len(frames) == 0 {
		fmt.Fprintln(r.out, "No stack frames available.")
		return
	}

	for i, frame := range frames {
		fmt.Fprintf(r.out, "#%d %s at %s\n", i, frame.Name, formatLocation(frame.Location))
	}
}

func (r *Renderer) Locals(variables []apidebug.Variable) {
	locals := make([]apidebug.Variable, 0, len(variables))
	params := make([]apidebug.Variable, 0, len(variables))

	for _, variable := range variables {
		if variable.Param {
			params = append(params, variable)
		} else {
			locals = append(locals, variable)
		}
	}

	if len(locals) == 0 && len(params) == 0 {
		fmt.Fprintln(r.out, "No local variables available.")
		return
	}

	if len(locals) > 0 {
		fmt.Fprintln(r.out, "Locals:")
		renderVariables(r.out, locals)
	}

	if len(params) > 0 {
		fmt.Fprintln(r.out, "Params:")
		renderVariables(r.out, params)
	}
}

func (r *Renderer) Evaluation(value apidebug.Value) {
	fmt.Fprintln(r.out, value.Display)
}

func (r *Renderer) Error(prefix string, err error) {
	if err == nil {
		return
	}

	fmt.Fprintf(r.out, "%s: %s\n", prefix, diagnostics.Format(err))
}

func (r *Renderer) snippet(location source.Range) {
	if location.SourceName != r.source.Name() || location.Line <= 0 {
		return
	}

	for _, snippet := range r.source.Snippet(location.Span) {
		if snippet.Line != location.Line {
			continue
		}

		lineNumber := strconv.Itoa(snippet.Line)
		fmt.Fprintf(r.out, "%s | %s\n", lineNumber, snippet.Text)

		if snippet.Caret != "" {
			fmt.Fprintf(r.out, "%s%s\n", strings.Repeat(" ", len(lineNumber)+3), snippet.Caret)
		}

		return
	}
}
