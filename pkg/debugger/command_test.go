package debugger

import (
	"strings"
	"testing"

	"github.com/MontFerret/ferret/v2"
)

func TestParseCommand(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		want   Command
		errHas string
	}{
		{name: "empty", input: "  ", want: Command{}},
		{
			name:  "break line",
			input: "break 12",
			want: Command{
				Name:              CommandBreak,
				Argument:          "12",
				Location:          ferret.DebugSourceLocation{Line: 12},
				BreakpointOptions: ferret.DebugBreakpointOptions{BindingMode: ferret.DebugBreakpointBindNextExecutableInFile},
			},
		},
		{
			name:  "break line and column",
			input: "break 12:4",
			want: Command{
				Name:              CommandBreak,
				Argument:          "12:4",
				Location:          ferret.DebugSourceLocation{Line: 12, Column: 4},
				BreakpointOptions: ferret.DebugBreakpointOptions{BindingMode: ferret.DebugBreakpointBindNextExecutableInFile},
			},
		},
		{
			name:  "break file and line",
			input: "break examples/demo.fql:12",
			want: Command{
				Name:              CommandBreak,
				Argument:          "examples/demo.fql:12",
				Location:          ferret.DebugSourceLocation{File: "examples/demo.fql", Line: 12},
				BreakpointOptions: ferret.DebugBreakpointOptions{BindingMode: ferret.DebugBreakpointBindNextExecutableInFile},
			},
		},
		{
			name:  "break file line and column",
			input: "break examples/demo.fql:12:4",
			want: Command{
				Name:              CommandBreak,
				Argument:          "examples/demo.fql:12:4",
				Location:          ferret.DebugSourceLocation{File: "examples/demo.fql", Line: 12, Column: 4},
				BreakpointOptions: ferret.DebugBreakpointOptions{BindingMode: ferret.DebugBreakpointBindNextExecutableInFile},
			},
		},
		{
			name:  "break Windows drive path",
			input: `break C:\work\demo.fql:12:4`,
			want: Command{
				Name:              CommandBreak,
				Argument:          `C:\work\demo.fql:12:4`,
				Location:          ferret.DebugSourceLocation{File: `C:\work\demo.fql`, Line: 12, Column: 4},
				BreakpointOptions: ferret.DebugBreakpointOptions{BindingMode: ferret.DebugBreakpointBindNextExecutableInFile},
			},
		},
		{
			name:  "break exact before location",
			input: "break --exact 12:4",
			want: Command{
				Name:              CommandBreak,
				Argument:          "--exact 12:4",
				Location:          ferret.DebugSourceLocation{Line: 12, Column: 4},
				BreakpointOptions: ferret.DebugBreakpointOptions{BindingMode: ferret.DebugBreakpointBindExact},
			},
		},
		{
			name:  "break next after location",
			input: "break 12 --next",
			want: Command{
				Name:              CommandBreak,
				Argument:          "12 --next",
				Location:          ferret.DebugSourceLocation{Line: 12},
				BreakpointOptions: ferret.DebugBreakpointOptions{BindingMode: ferret.DebugBreakpointBindNextExecutableInFile},
			},
		},
		{
			name:  "break in function",
			input: "break --in-function demo.fql:12",
			want: Command{
				Name:              CommandBreak,
				Argument:          "--in-function demo.fql:12",
				Location:          ferret.DebugSourceLocation{File: "demo.fql", Line: 12},
				BreakpointOptions: ferret.DebugBreakpointOptions{BindingMode: ferret.DebugBreakpointBindNextExecutableInFunction},
			},
		},
		{name: "delete", input: "delete 1", want: Command{Name: CommandDelete, Argument: "1", BreakpointID: ferret.DebugBreakpointID(1)}},
		{name: "print expression", input: "print users[0].name + \" value\"", want: Command{Name: CommandPrint, Argument: `users[0].name + " value"`}},
		{name: "help", input: "help", want: Command{Name: CommandHelp}},
		{name: "breakpoints", input: "breakpoints", want: Command{Name: CommandBreakpoints}},
		{name: "continue", input: "continue", want: Command{Name: CommandContinue}},
		{name: "step", input: "step", want: Command{Name: CommandStep}},
		{name: "next", input: "next", want: Command{Name: CommandNext}},
		{name: "out", input: "out", want: Command{Name: CommandOut}},
		{name: "pause", input: "pause", want: Command{Name: CommandPause}},
		{name: "where", input: "where", want: Command{Name: CommandWhere}},
		{name: "locals", input: "locals", want: Command{Name: CommandLocals}},
		{name: "quit", input: "quit", want: Command{Name: CommandQuit}},
		{name: "tab whitespace", input: "p\tuser.name", want: Command{Name: CommandPrint, Argument: "user.name"}},
		{
			name:  "break alias",
			input: "b 4",
			want: Command{
				Name:              CommandBreak,
				Argument:          "4",
				Location:          ferret.DebugSourceLocation{Line: 4},
				BreakpointOptions: ferret.DebugBreakpointOptions{BindingMode: ferret.DebugBreakpointBindNextExecutableInFile},
			},
		},
		{name: "continue alias", input: "c", want: Command{Name: CommandContinue}},
		{name: "step alias", input: "s", want: Command{Name: CommandStep}},
		{name: "next alias", input: "n", want: Command{Name: CommandNext}},
		{name: "where alias", input: "bt", want: Command{Name: CommandWhere}},
		{name: "quit alias", input: "q", want: Command{Name: CommandQuit}},
		{name: "invalid command", input: "wat", errHas: "unknown command: wat"},
		{name: "missing break line", input: "break", errHas: "usage: break"},
		{name: "invalid break line", input: "break file.fql:nope", errHas: "usage: break"},
		{name: "zero break line", input: "break 0", errHas: "usage: break"},
		{name: "zero break column", input: "break 12:0", errHas: "usage: break"},
		{name: "unknown break option", input: "break --nearest 12", errHas: "unknown break option"},
		{name: "conflicting break options", input: "break --exact 12 --next", errHas: "mutually exclusive"},
		{name: "duplicate break option", input: "break --exact 12 --exact", errHas: "mutually exclusive"},
		{name: "missing location after option", input: "break --exact", errHas: "usage: break"},
		{name: "extra break argument", input: "break 12 extra", errHas: "usage: break"},
		{name: "path with whitespace", input: "break my file.fql:12", errHas: "usage: break"},
		{name: "missing delete id", input: "delete", errHas: "usage: delete"},
		{name: "missing print expression", input: "print", errHas: "usage: print"},
		{name: "unexpected argument", input: "continue now", errHas: "continue does not accept arguments"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ParseCommand(test.input)
			if test.errHas != "" {
				if err == nil || !strings.Contains(err.Error(), test.errHas) {
					t.Fatalf("expected error containing %q, got %v", test.errHas, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("unexpected command: got %#v, want %#v", got, test.want)
			}
		})
	}
}
