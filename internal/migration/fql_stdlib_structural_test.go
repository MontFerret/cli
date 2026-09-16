package migration

import (
	"strconv"
	"strings"
	"testing"

	"github.com/MontFerret/ferret/v2/pkg/source"
)

func TestMigrateFQLStdlibArgumentRules(t *testing.T) {
	tests := []struct{ before, after string }{
		{"first(a)", "arrays::first(a)"},
		{"flatten(a)", "arrays::flatten(a)"},
		{"flatten(a, 2)", "arrays::flatten(a, 2)"},
		{"last(a)", "arrays::last(a)"},
		{"sorted(a)", "arrays::sorted(a)"},
		{"unique(a)", "arrays::unique(a)"},
		{"append(a, v)", "arrays::append(a, v)"},
		{"push(a, v)", "arrays::append(a, v)"},
		{"nth(a, 1)", "arrays::at(a, 1)"},
		{"remove_value(a, v)", "arrays::remove(a, v)"},
		{"remove_values(a, b)", "arrays::remove_any(a, b)"},
		{"slice(a, 1)", "arrays::slice(a, 1)"},
		{"slice(a, 1, 2)", "arrays::slice(a, 1, 2)"},
		{"intersection(a, b)", "arrays::intersection(a, b)"},
		{"minus(a, b)", "arrays::difference(a, b)"},
		{"union(a, b)", "arrays::concat(a, b)"},
		{"union_distinct(a, b)", "arrays::union(a, b)"},
		{"position(a, v)", "arrays::contains(a, v)"},
		{"position(a, v, false)", "arrays::contains(a, v)"},
		{"position(a, v, true)", "arrays::index_of(a, v)"},
		{"position(a, v, ((TRUE)))", "arrays::index_of(a, v)"},
		{"position(a, v, FaLsE)", "arrays::contains(a, v)"},
		{"append(a, v, false)", "arrays::append(a, v)"},
		{"push(a, v, (FALSE))", "arrays::append(a, v)"},
		{"remove_value(a, v, -1)", "arrays::remove(a, v)"},
		{"remove_value(a, v, -27)", "arrays::remove(a, v)"},
		{"remove_value(a, v, ((-(2))))", "arrays::remove(a, v)"},
		{"remove_value(a, v, -" + strconv.Itoa(int(^uint(0)>>1)) + ")", "arrays::remove(a, v)"},
		{"outersection(a, b)", "arrays::symmetric_difference(a, b)"},
		{"sorted_unique(a)", "arrays::sorted(arrays::unique(a))"},
		{"shift(a)", "arrays::slice(a, 1)"},
		{"keys(o, false)", "object::keys(o)"},
		{"keys(o, ((true)))", "arrays::sorted(object::keys(o))"},
		{"date_diff(a, b, unit, true)", "datetime::diff(a, b, unit)"},
		{"date_diff(a, b, unit, (TrUe))", "datetime::diff(a, b, unit)"},
		{"range(a, b)", "arrays::range(a, b)"},
		{"range(a, b, step)", "arrays::range(a, b, step)"},
		{"rand()", "random::float()"},
		{"position(a, v, false,)", "arrays::contains(a, v)"},
		{"keys(o, true,)", "arrays::sorted(object::keys(o))"},
		{"shift(a,)", "arrays::slice(a, 1)"},
		{"sorted_unique(a,)", "arrays::sorted(arrays::unique(a))"},
	}

	for _, test := range tests {
		name, rest, _ := strings.Cut(test.before, "(")
		for _, spelling := range []string{name, strings.ToUpper(name), strings.ToUpper(name[:1]) + name[1:]} {
			t.Run(spelling+"("+rest, func(t *testing.T) {
				input := "return " + spelling + "(" + rest
				assertFQLStdlibMigration(t, input, "return "+test.after, 0)

				diagnostics := checkStdlibTestSource(t, input)
				if len(diagnostics) != 1 || !strings.Contains(diagnostics[0].Message, "should use") ||
					diagnostics[0].Line != 1 || diagnostics[0].Column != 8 {
					t.Fatalf("unexpected automatic check finding: %#v", diagnostics)
				}

				if got := checkStdlibTestSource(t, "return "+test.after); len(got) != 0 {
					t.Fatalf("canonical expression has findings: %#v", got)
				}
			})
		}
	}
}

func TestMigrateFQLStdlibManualModes(t *testing.T) {
	tests := []struct{ call, reason string }{
		{"position(a, v, mode)", "Boolean or an index"},
		{"append(a, v, true)", "existing duplicates"},
		{"push(a, v, true)", "existing duplicates"},
		{"append(a, v, mode)", "dynamic flag"},
		{"push(a, v, mode)", "dynamic flag"},
		{"remove_value(a, v, 0)", "no limit mode"},
		{"remove_value(a, v, 2)", "no limit mode"},
		{"remove_value(a, v, limit)", "negative integer literal"},
		{"outersection(a, b, c)", "exactly one input"},
		{"outersection(a, b, c, d)", "odd-number-of-inputs"},
		{"keys(o, mode)", "dynamic sort mode"},
		{"date_diff(a, b, unit)", "truncates toward zero"},
		{"date_diff(a, b, unit, false)", "always returns a Float"},
		{"date_diff(a, b, unit, mode)", "literal true"},
		{"rand(10)", "max/2 to max*2"},
		{"rand(10, 2)", "rounded/floored"},
		{"pop(a)", "evaluation count"},
		{"unshift(a, v)", "evaluation order"},
		{"remove_nth(a, 1)", "copied host list"},
	}

	for _, call := range []string{"position(a)", "position(a, v, true, extra)", "append()", "push(a)", "append(a, v, false, extra)", "remove_value(a)", "remove_value(a, v, -1, extra)", "keys()", "keys(o, true, extra)", "date_diff(a)", "date_diff(a, b, unit, true, extra)", "range(a)", "range(a, b, step, extra)", "rand(1, 2, 3)"} {
		tests = append(tests, struct{ call, reason string }{call, "arity"})
	}

	for _, call := range []string{"sorted_unique()", "sorted_unique(a, b)", "shift()", "shift(a, b)"} {
		tests = append(tests, struct{ call, reason string }{call, "exactly one"})
	}

	for _, call := range []string{"outersection()", "outersection(a)"} {
		tests = append(tests, struct{ call, reason string }{call, "at least two"})
	}

	for _, mode := range []string{"-0", "-0.0", "-1.0", "-1e0", "-(1 + 1)", "-limit", "-(-2)", "1 - 2", "(-2)?", "-999999999999999999999999999999"} {
		tests = append(tests, struct{ call, reason string }{"remove_value(a, v, " + mode + ")", "negative integer literal"})
	}

	for _, mode := range []string{"true == true", "!false", "flag", "1", "\"true\"", "(true)?", "(true) ON ERROR RETURN false", "true ? true : false"} {
		tests = append(tests, struct{ call, reason string }{"position(a, v, " + mode + ")", "dynamic mode"})
	}

	for _, test := range tests {
		t.Run(test.call, func(t *testing.T) {
			input := "// original location\nRETURN   " + test.call
			result := assertFQLStdlibMigration(t, input, input, 1)
			diagnostics := checkStdlibTestSource(t, input)
			if len(diagnostics) != 1 || diagnostics[0].Help != result.ManualActions[0].Reason ||
				!strings.Contains(diagnostics[0].Help, test.reason) || diagnostics[0].Line != 2 || diagnostics[0].Column != 10 {
				t.Fatalf("check/run manual findings disagree: %#v, %#v", diagnostics, result.ManualActions)
			}
		})
	}
}

func TestMigrateFQLStdlibNestedStructuralRules(t *testing.T) {
	tests := []struct{ input, want string }{
		{"return sorted_unique(nth(values, 1))", "return arrays::sorted(arrays::unique(arrays::at(values, 1)))"},
		{"return keys(json_parse(body), true)", "return arrays::sorted(object::keys(encoding::json_parse(body)))"},
		{"return shift(union(a, b))", "return arrays::slice(arrays::concat(a, b), 1)"},
		{"return sorted_unique(shift(sorted_unique(a)))", "return arrays::sorted(\n    arrays::unique(arrays::slice(arrays::sorted(arrays::unique(a)), 1))\n)"},
		{"return [shift(a), keys(o, true), position(a, v, false)]", "return [\n    arrays::slice(a, 1),\n    arrays::sorted(object::keys(o)),\n    arrays::contains(a, v)\n]"},
		{"for x in shift(a) return sorted_unique(x)", "return for x in arrays::slice(a, 1) {\n    return arrays::sorted(arrays::unique(x))\n}"},
		{"for x in sorted_unique(a) { return shift(x) }", "return for x in arrays::sorted(arrays::unique(a)) {\n    return arrays::slice(x, 1)\n}"},
		{"return sorted_unique(nth(a, 1)?)?", "return arrays::sorted(arrays::unique(arrays::at(a, 1)?))?"},
		{"return sorted_unique(a) ON ERROR RETURN []", "return arrays::sorted(arrays::unique(a)) on error return []"},
		{"return shift(\n    union(a, b),\n)", "return arrays::slice(arrays::concat(a, b), 1)"},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			assertFQLStdlibMigration(t, test.input, test.want, 0)
		})
	}

	assertFQLStdlibMigration(t, "return append(shift(a), v, true)", "return append(arrays::slice(a, 1), v, true)", 1)
}

func TestFQLStdlibStructuralEditsPreserveTrivia(t *testing.T) {
	tests := []struct{ input, want string }{
		{"return keys(\n    obj /* important */,\n    (/* mode */ true /* end */),\n)", "return arrays::sorted(object::keys(\n    obj /* important */\n    /* mode */  /* end */,\n))"},
		{"return remove_value(a, v, (/* group */ - /* sign */ 2 /* end */))", "return arrays::remove(a, v /* group */  /* sign */  /* end */)"},
		{"return shift(\n    union(a, b), // keep λ\n)", "return arrays::slice(\n    arrays::concat(a, b), // keep λ\n1)"},
		{"return sorted_unique /* call */ (nth(values, 1))?", "return arrays::sorted(arrays::unique /* call */ (arrays::at(values, 1)))?"},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			src := source.New("query.fql", test.input)
			program, err := parseFQLSource(src)
			if err != nil {
				t.Fatal(err)
			}

			edits, actions, err := planFQLStdlib(src, program)
			if err != nil || len(actions) != 0 {
				t.Fatalf("plan edits: %v, %#v", err, actions)
			}

			got, err := applyFQLEdits(test.input, edits)
			if err != nil || got != test.want {
				t.Fatalf("raw edits = %q, want %q; err=%v", got, test.want, err)
			}

			result, err := migrateFQLSource(src)
			if err != nil {
				if !strings.Contains(err.Error(), "preserve comments") || result.Changed || result.Data != nil {
					t.Fatalf("unsafe formatter result: %#v, %v", result, err)
				}
			} else {
				formatted, err := parseFQLSource(source.New("query.fql", string(result.Data)))
				if err != nil || strings.Join(fqlComments(formatted), "\n") != strings.Join(fqlComments(program), "\n") {
					t.Fatalf("formatted comments changed: %s, %v", result.Data, err)
				}
			}
		})
	}
}
