package migration

import (
	"strings"
	"testing"

	"github.com/MontFerret/ferret/v2/pkg/source"
)

func TestFQLStdlibStructuralResolutionGuards(t *testing.T) {
	tests := []struct{ header, call, reason string }{
		{"func SHIFT(a) { return a }", "shift(a)", "file-local"},
		{"use custom::calculate as position", "POSITION(a, v, true)", "function alias"},
		{"use custom as arrays", "keys(o, true)", "arrays"},
		{"use custom as object", "keys(o, true)", "object"},
		{"use custom as arrays", "sorted_unique(a)", "arrays"},
		{"use custom as arrays", "shift(a)", "arrays"},
		{"use custom as arrays", "position(a, v, true)", "arrays"},
		{"use custom as arrays", "range(1, 2)", "arrays"},
		{"use custom as random", "rand()", "random"},
		{"use custom::calculate as random", "rand()", "random"},
		{"use custom as datetime", "date_diff(a, b, unit, true)", "datetime"},
		{"use custom as arrays\nuse arrays as arrays", "keys(o, true)", "arrays"},
	}

	for _, test := range tests {
		t.Run(test.header+"/"+test.call, func(t *testing.T) {
			input := test.header + "\nreturn " + test.call
			result := assertFQLStdlibMigration(t, input, input, 1)
			diagnostics := checkStdlibTestSource(t, input)
			if len(diagnostics) != 1 || diagnostics[0].Help != result.ManualActions[0].Reason ||
				!strings.Contains(diagnostics[0].Help, test.reason) {
				t.Fatalf("collision not shared by check/run: %#v, %#v", diagnostics, result.ManualActions)
			}
		})
	}

	for _, header := range []string{"use arrays as arrays", "use custom as ARRAYS", "use custom as unrelated"} {
		assertFQLStdlibMigration(t, header+"\nreturn keys(o, true)", header+"\nreturn arrays::sorted(object::keys(o))", 0)
	}

	assertFQLStdlibMigration(t, "use custom as arrays\nreturn keys(o, false)", "use custom as arrays\nreturn object::keys(o)", 0)
	assertFQLStdlibMigration(t, "use custom as random\nreturn random::float()", "use custom as random\nreturn random::float()", 0)
	assertFQLStdlibMigration(t, "use custom as arrays\nreturn keys(json_parse(body), true)", "use custom as arrays\nreturn keys(encoding::json_parse(body), true)", 1)
}

func TestFQLStdlibStructuralDiagnosticDescriptions(t *testing.T) {
	for _, test := range []struct{ call, description string }{
		{"sorted_unique(a)", "arrays::sorted(arrays::unique(...))"},
		{"keys(o, true)", "arrays::sorted(object::keys(...))"},
		{"shift(a)", "arrays::slice(..., 1)"},
	} {
		diagnostics := checkStdlibTestSource(t, "return "+test.call)
		if len(diagnostics) != 1 || !strings.Contains(diagnostics[0].Message, "`"+test.description+"`") {
			t.Fatalf("missing composed replacement: %#v", diagnostics)
		}
	}

	diagnostics := checkStdlibTestSource(t, "return position(a, v, true)")
	if len(diagnostics) != 1 || !strings.Contains(diagnostics[0].Help, "final literal mode argument is removed") {
		t.Fatalf("missing argument-removal guidance: %#v", diagnostics)
	}
}

func TestFQLStdlibStructuralFailurePreservesManualFindings(t *testing.T) {
	input := "let a = append(items, v, true)\nlet b = rand(10)\nreturn keys /* preserve me */ (obj, true)"
	result, err := migrateFQLSource(source.New("query.fql", input))
	if err == nil || !strings.Contains(err.Error(), "preserve comments") || result.Changed || result.Data != nil {
		t.Fatalf("unsafe formatting accepted: %#v, %v", result, err)
	}

	if len(result.ManualActions) != 2 || result.ManualActions[0].Detail != "append(...)" ||
		result.ManualActions[0].Line != 1 || result.ManualActions[1].Detail != "rand(...)" || result.ManualActions[1].Line != 2 {
		t.Fatalf("lost semantic findings: %#v", result.ManualActions)
	}
}
