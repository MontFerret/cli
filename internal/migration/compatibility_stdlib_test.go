package migration

import (
	"context"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/MontFerret/ferret/v2/pkg/source"
)

func TestCheckFQLStdlibReplacements(t *testing.T) {
	tests := []struct{ call, name, target string }{
		{`has({ foo: "bar" }, "baz")`, "has", "object::has_key"},
		{`JSON_PARSE("{}")`, "JSON_PARSE", "encoding::json_parse"},
		{`Sha1(text)`, "Sha1", "crypto::sha1"},
		{`base(path)`, "base", "path::base"},
		{`keys(obj)`, "keys", "object::keys"},
		{`Date_Year(dt)`, "Date_Year", "datetime::year"},
		{`ABS(-1)`, "ABS", "math::abs"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			diagnostics := checkStdlibTestSource(t, "return "+test.call)
			want := []CompatibilityDiagnostic{{
				Path:    "query.fql",
				Message: fmt.Sprintf("Legacy stdlib call `%s` should use `%s`.", test.name, test.target),
				Help:    "Preview automatic replacements with `ferret migrate run --print`.",
				Line:    1,
				Column:  8,
				Kind:    CompatibilityDiagnosticIssue,
			}}
			if !reflect.DeepEqual(diagnostics, want) {
				t.Fatalf("diagnostics = %#v, want %#v", diagnostics, want)
			}

			canonical := "return " + test.target + strings.TrimPrefix(test.call, test.name)
			if diagnostics := checkStdlibTestSource(t, canonical); len(diagnostics) != 0 {
				t.Fatalf("canonical source has findings: %#v", diagnostics)
			}
		})
	}
}

func TestCheckFQLStdlibManualReview(t *testing.T) {
	tests := []struct{ call, reason string }{
		{`join("a", "b")`, "legacy path::join"},
		{`join(["a", "b"], ",")`, "modern global string joining"},
		{"keys()", "only keys(obj)"},
		{"keys(obj, true)", "argument-aware"},
		{"keys(obj, false)", "argument-aware"},
		{"keys(obj, option)", "argument-aware"},
		{"keys(obj, true, extra)", "argument-aware"},
		{`date_compare(a, b, "year")`, "component-range"},
		{`date_diff(a, b, "day")`, "integer/floating"},
		{"average(xs)", "heterogeneous"},
		{"sum(xs)", "heterogeneous"},
		{"min(xs)", "heterogeneous"},
		{"max(xs)", "heterogeneous"},
		{"median(xs)", "heterogeneous"},
		{"percentile(xs, 50)", "heterogeneous"},
		{"stddev_population(xs)", "heterogeneous"},
		{"stddev_sample(xs)", "heterogeneous"},
		{"variance_population(xs)", "heterogeneous"},
		{"variance_sample(xs)", "heterogeneous"},
	}

	for _, test := range tests {
		t.Run(test.call, func(t *testing.T) {
			input := "// original location\nRETURN   " + test.call
			diagnostics := checkStdlibTestSource(t, input)

			name, _, _ := strings.Cut(test.call, "(")
			if len(diagnostics) != 1 || diagnostics[0].Path != "query.fql" ||
				diagnostics[0].Kind != CompatibilityDiagnosticIssue ||
				diagnostics[0].Line != 2 || diagnostics[0].Column != 10 ||
				diagnostics[0].Message != "Stdlib call `"+name+"` needs manual review." ||
				!strings.Contains(diagnostics[0].Help, test.reason) {
				t.Fatalf("unexpected manual finding: %#v", diagnostics)
			}

			migration, err := migrateFQLSource(source.New("query.fql", input))
			if err != nil {
				t.Fatal(err)
			}

			if len(migration.ManualActions) != 1 || migration.ManualActions[0].Reason != diagnostics[0].Help {
				t.Fatalf("check and run disagree: diagnostics=%#v actions=%#v", diagnostics, migration.ManualActions)
			}
		})
	}
}

func TestCheckFQLStdlibResolutionGuards(t *testing.T) {
	tests := []struct{ input, reason string }{
		{"func ABS(x) { return x }\nreturn Abs(-1)", "file-local"},
		{"let value = abs(-1)\nfunc abs(x) { return x }\nreturn value", "file-local"},
		{"func outer() { func abs(x) { return x } return 1 }\nreturn abs(-1)", "file-local"},
		{"use custom::calculate as abs\nreturn ABS(-1)", "function alias"},
		{"use custom as math\nreturn abs(-1)", "redirect the replacement math::abs"},
		{"use custom::calculate as math\nreturn abs(-1)", "redirect the replacement math::abs"},
		{"use custom as math\nuse math as math\nreturn abs(-1)", "redirect"},
		{"func average(xs) { return xs }\nreturn average(xs)", "file-local"},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			diagnostics := checkStdlibTestSource(t, test.input)
			if len(diagnostics) != 1 || !strings.Contains(diagnostics[0].Message, "needs manual review") ||
				!strings.Contains(diagnostics[0].Help, test.reason) {
				t.Fatalf("unexpected collision finding: %#v", diagnostics)
			}
		})
	}

	for _, header := range []string{
		"use custom as other",
		"use custom as abs",
		"use custom::calculate as other",
		"use math as math",
		"use custom as MATH",
	} {
		diagnostics := checkStdlibTestSource(t, header+"\nreturn abs(-1)")
		if len(diagnostics) != 1 || diagnostics[0].Message != "Legacy stdlib call `abs` should use `math::abs`." {
			t.Fatalf("unrelated alias suppressed replacement: %#v", diagnostics)
		}
	}
}

func TestCheckFQLStdlibIgnoresUnrelatedSource(t *testing.T) {
	input := `use custom as math
// abs(-1) and average(xs)
let has = "json_parse()"
return { has, abs: "date_diff()", values: [
    math::abs(-1), CUSTOM::HAS(obj, "key"), object::keys(obj),
    union(a, b), push(a, 1), nth(a, 0), rand(), range(1, 10)
] }`
	if diagnostics := checkStdlibTestSource(t, input); len(diagnostics) != 0 {
		t.Fatalf("unrelated source has findings: %#v", diagnostics)
	}
}

func TestCheckFQLCompatibilityCombinesStdlibAndLoopFindings(t *testing.T) {
	root := writeCompatibilityFixture(t, map[string]string{
		"a_broken.fql": "return [average(xs),",
		"b_mixed.fql": `let label = "日本語"
for item in values(obj)
    return ["λ", has(item, "key"), average([abs(-1)]), average(item)]`,
		"c_comments.fql":  `return [average(xs), sha1 /* preserve me */ (text)]`,
		"d_collision.fql": "func ABS(x) { return x }\nreturn [abs(-1), sha1(text)]",
		"go.mod":          "module example.com/no-toolchain\n",
	})
	path := filepath.Join(root, "b_mixed.fql")
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}

	before := compatibilityFixtureContents(t, root)
	var previous *CompatibilityResult
	for range 2 {
		// No migration planner or Go runner is configured: check only inspects FQL.
		result, err := new(Migrator).CheckCompatibility(context.Background(), CompatibilityOptions{Path: root})
		if err != nil {
			t.Fatal(err)
		}

		if result.ScannedFiles != 4 || len(result.Diagnostics) != 11 {
			t.Fatalf("unexpected combined findings: %#v", result)
		}

		broken := result.Diagnostics[0]
		if broken.Kind != CompatibilityDiagnosticFailure || !strings.HasPrefix(broken.Message, "Could not check v1 compatibility:") {
			t.Fatalf("unexpected parse failure: %#v", broken)
		}

		want := []struct {
			file, message string
			line, column  int
		}{
			{"b_mixed.fql", finalForMessage, 2, 1},
			{"b_mixed.fql", "Legacy stdlib call `values` should use `object::values`.", 2, 13},
			{"b_mixed.fql", "Legacy stdlib call `has` should use `object::has_key`.", 3, 19},
			{"b_mixed.fql", "Stdlib call `average` needs manual review.", 3, 37},
			{"b_mixed.fql", "Legacy stdlib call `abs` should use `math::abs`.", 3, 46},
			{"b_mixed.fql", "Stdlib call `average` needs manual review.", 3, 57},
			{"c_comments.fql", "Stdlib call `average` needs manual review.", 1, 9},
			{"c_comments.fql", "Legacy stdlib call `sha1` should use `crypto::sha1`.", 1, 22},
			{"d_collision.fql", "Stdlib call `abs` needs manual review.", 2, 9},
			{"d_collision.fql", "Legacy stdlib call `sha1` should use `crypto::sha1`.", 2, 18},
		}
		for index, expected := range want {
			diagnostic := result.Diagnostics[index+1]
			if diagnostic.Path != compatibilityTestDisplayPath(t, filepath.Join(root, expected.file)) ||
				diagnostic.Kind != CompatibilityDiagnosticIssue || diagnostic.Message != expected.message ||
				diagnostic.Line != expected.line || diagnostic.Column != expected.column {
				t.Fatalf("diagnostic %d = %#v, want %#v", index+1, diagnostic, expected)
			}
		}

		if previous != nil && !reflect.DeepEqual(result, previous) {
			t.Fatalf("retry changed diagnostics: before=%#v after=%#v", previous, result)
		}

		previous = result

		if after := compatibilityFixtureContents(t, root); !maps.Equal(before, after) {
			t.Fatal("check modified files")
		}

		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}

		if info.Mode().Perm() != 0o600 {
			t.Fatalf("check changed file mode: %v", info.Mode())
		}
	}
}

func BenchmarkCheckFQLStdlibCompatibility(b *testing.B) {
	for _, test := range []struct {
		name, input string
		findings    int
	}{
		{"legacy", "return [json_parse(text), sha1(text), base(path), has(obj, key), date_year(dt), abs(-1), average(xs)]", 700},
		{"canonical", "return [encoding::json_parse(text), crypto::sha1(text), path::base(path), object::has_key(obj, key), datetime::year(dt), math::abs(-1)]", 0},
	} {
		b.Run(test.name, func(b *testing.B) {
			root := b.TempDir()
			for index := range 100 {
				path := filepath.Join(root, fmt.Sprintf("query_%03d.fql", index))
				if err := os.WriteFile(path, []byte(test.input), 0o644); err != nil {
					b.Fatal(err)
				}
			}

			b.ReportAllocs()
			b.ResetTimer()

			for range b.N {
				result, err := new(Migrator).CheckCompatibility(context.Background(), CompatibilityOptions{Path: root})
				if err != nil {
					b.Fatal(err)
				}

				if result.ScannedFiles != 100 || len(result.Diagnostics) != test.findings {
					b.Fatalf("unexpected benchmark findings: %#v", result)
				}
			}
		})
	}
}

func checkStdlibTestSource(t *testing.T, input string) []CompatibilityDiagnostic {
	t.Helper()

	src := source.New("query.fql", input)
	program, err := parseFQLSource(src)
	if err != nil {
		t.Fatal(err)
	}

	diagnostics, err := checkFQLStdlib(src, program)
	if err != nil {
		t.Fatal(err)
	}

	return diagnostics
}
