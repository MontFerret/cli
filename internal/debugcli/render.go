package debugcli

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/MontFerret/ferret/v2"
	"github.com/MontFerret/ferret/v2/pkg/diagnostics"
	"github.com/MontFerret/ferret/v2/pkg/source"
)

const helpText = `Commands:
  break <location>              Set at next executable location in file
  break --exact <location>      Set only at the exact executable location
  break --next <location>       Set at next executable location in file
  break --in-function <location> Set at next executable location in function
  breakpoints                   List breakpoints
  delete <id>                   Delete breakpoint
  continue                      Resume execution
  step                          Step into next source location
  next                          Step over current source location
  out                           Step out of current frame
  pause                         Pause at the next source location
  where                         Show stack trace
  locals                        Show local variables
  print <expr>                  Evaluate a safe expression (no calls, queries, or mutation)
  quit                          Stop debugging and exit

Locations: 12, 12:4, file.fql:12, file.fql:12:4

Aliases: b=break, c=continue, s=step, n=next, bt=where, p=print, q=quit`

type Renderer struct {
	out    io.Writer
	source *source.Source
}

func NewRenderer(out io.Writer, src *source.Source) *Renderer {
	return &Renderer{out: out, source: src}
}

func (r *Renderer) Help() {
	fmt.Fprintln(r.out, helpText)
}

func (r *Renderer) Event(event *ferret.DebugEvent) {
	if event == nil {
		fmt.Fprintln(r.out, "Debugger returned no event.")
		return
	}

	switch event.Reason {
	case ferret.DebugReasonEntry:
		fmt.Fprintf(r.out, "Paused at %s\n", formatLocation(event.Location))
		r.snippet(event.Location)
	case ferret.DebugReasonBreakpoint:
		switch len(event.HitBreakpointIDs) {
		case 0:
			fmt.Fprintf(r.out, "Paused on breakpoint at %s\n", formatLocation(event.Location))
		case 1:
			fmt.Fprintf(r.out, "Paused on breakpoint %d at %s\n", event.HitBreakpointIDs[0], formatLocation(event.Location))
		default:
			fmt.Fprintf(r.out, "Paused on breakpoints %s at %s\n", formatBreakpointIDs(event.HitBreakpointIDs), formatLocation(event.Location))
		}
		r.snippet(event.Location)
	case ferret.DebugReasonStep:
		fmt.Fprintf(r.out, "Paused after step at %s\n", formatLocation(event.Location))
		r.snippet(event.Location)
	case ferret.DebugReasonPause:
		fmt.Fprintf(r.out, "Paused on pause request at %s\n", formatLocation(event.Location))
		r.snippet(event.Location)
	case ferret.DebugReasonRuntimeError:
		fmt.Fprintln(r.out, "Paused on runtime error.")
		r.error(event.Error)
	case ferret.DebugReasonCompleted:
		fmt.Fprintln(r.out, "Program completed.")
		if event.Output != nil {
			fmt.Fprintln(r.out, "Result:")
			fmt.Fprintln(r.out, string(event.Output.Content))
		}
	case ferret.DebugReasonTerminated:
		fmt.Fprintln(r.out, "Program terminated.")
		r.error(event.Error)
	default:
		fmt.Fprintf(r.out, "Debugger stopped: %s\n", event.Reason)
	}
}

func (r *Renderer) BreakpointSet(breakpoint ferret.DebugBreakpoint) {
	requested := formatSourceLocation(breakpoint.File, breakpoint.RequestedLine, breakpoint.RequestedColumn)
	mode := formatBindingMode(breakpoint.BindingMode)
	if !breakpoint.Bound {
		fmt.Fprintf(r.out, "Breakpoint %d could not be bound at %s (%s).\n", breakpoint.ID, requested, mode)
		return
	}

	bound := formatSourceLocation(breakpoint.File, breakpoint.Line, breakpoint.Column)
	if requested == bound {
		fmt.Fprintf(r.out, "Breakpoint %d set at %s (%s).\n", breakpoint.ID, bound, mode)
		return
	}
	fmt.Fprintf(r.out, "Breakpoint %d set at %s (requested %s, %s).\n", breakpoint.ID, bound, requested, mode)
}

func (r *Renderer) Breakpoints(breakpoints []ferret.DebugBreakpoint) {
	if len(breakpoints) == 0 {
		fmt.Fprintln(r.out, "No breakpoints.")
		return
	}

	table := tabwriter.NewWriter(r.out, 0, 4, 2, ' ', 0)
	fmt.Fprintln(table, "ID\tRequested\tBound\tMode\tState")

	for _, breakpoint := range breakpoints {
		requested := formatSourceLocation(breakpoint.File, breakpoint.RequestedLine, breakpoint.RequestedColumn)
		bound := "-"
		state := "unbound"
		if breakpoint.Bound {
			bound = formatSourceLocation(breakpoint.File, breakpoint.Line, breakpoint.Column)
			state = "bound"
		}
		fmt.Fprintf(table, "%d\t%s\t%s\t%s\t%s\n", breakpoint.ID, requested, bound, formatBindingMode(breakpoint.BindingMode), state)
	}

	_ = table.Flush()
}

func (r *Renderer) Frames(frames []ferret.DebugFrame) {
	if len(frames) == 0 {
		fmt.Fprintln(r.out, "No stack frames available.")
		return
	}

	for i, frame := range frames {
		fmt.Fprintf(r.out, "#%d %s at %s\n", i, frame.Name, formatLocation(frame.Location))
	}
}

func (r *Renderer) Locals(variables []ferret.DebugVariable) {
	locals := make([]ferret.DebugVariable, 0, len(variables))
	params := make([]ferret.DebugVariable, 0, len(variables))
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

func (r *Renderer) Evaluation(value ferret.DebugValue) {
	fmt.Fprintln(r.out, value.Display)
}

func (r *Renderer) Error(prefix string, err error) {
	if err == nil {
		return
	}
	fmt.Fprintf(r.out, "%s: %s\n", prefix, err)
}

func (r *Renderer) error(err error) {
	if err != nil {
		fmt.Fprintln(r.out, diagnostics.Format(err))
	}
}

func (r *Renderer) snippet(location ferret.DebugLocation) {
	if r.source == nil || location.File != r.source.Name() || location.Line <= 0 {
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

func renderVariables(out io.Writer, variables []ferret.DebugVariable) {
	for _, variable := range variables {
		fmt.Fprintf(out, "  %s = %s\n", variable.Name, variable.Value.Display)
	}
}

func formatBreakpointIDs(ids []ferret.DebugBreakpointID) string {
	values := make([]string, 0, len(ids))
	for _, id := range ids {
		values = append(values, strconv.Itoa(int(id)))
	}
	return strings.Join(values, ", ")
}

func formatBindingMode(mode ferret.DebugBreakpointBindingMode) string {
	switch mode {
	case ferret.DebugBreakpointBindExact:
		return "exact"
	case ferret.DebugBreakpointBindNextExecutableInFunction:
		return "in-function"
	default:
		return "next-file"
	}
}

func formatSourceLocation(file string, line, column int) string {
	return formatLocation(ferret.DebugLocation{File: file, Line: line, Column: column})
}

func formatLocation(location ferret.DebugLocation) string {
	if location.Column > 0 {
		return fmt.Sprintf("%s:%d:%d", location.File, location.Line, location.Column)
	}
	if location.Line > 0 {
		return fmt.Sprintf("%s:%d", location.File, location.Line)
	}
	return location.File
}
