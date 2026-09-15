package migration

import (
	"context"
	"strings"
	"testing"

	ferret "github.com/MontFerret/ferret/v2"
	"github.com/MontFerret/ferret/v2/pkg/source"
	"github.com/MontFerret/ferret/v2/pkg/stdlib"
)

func TestMigrateFQLStdlibMappings(t *testing.T) {
	// These expectations are independent of the registry so omissions and wrong
	// destinations cannot make the tests pass by changing both input and expectation.
	tests := []struct{ legacy, canonical, args string }{
		{"json_parse", "encoding::json_parse", `"{}"`},
		{"json_stringify", "encoding::json_stringify", "obj"},
		{"encode_uri_component", "encoding::query_escape", "text"},
		{"decode_uri_component", "encoding::query_unescape", "text"},
		{"to_base64", "encoding::base64_encode", "text"},
		{"from_base64", "encoding::base64_decode", "text"},
		{"escape_html", "encoding::html_escape", "text"},
		{"unescape_html", "encoding::html_unescape", "text"},
		{"md5", "crypto::md5", "text"},
		{"sha1", "crypto::sha1", "text"},
		{"sha512", "crypto::sha512", "text"},
		{"random_token", "crypto::random_token", "16"},
		{"base", "path::base", "path"},
		{"clean", "path::clean", "path"},
		{"dir", "path::dir", "path"},
		{"ext", "path::ext", "path"},
		{"is_abs", "path::is_abs", "path"},
		{"separate", "path::separate", "path"},
		{"match", "path::match", "pattern, path"},
		{"values", "object::values", "obj"},
		{"has", "object::has_key", `obj, "key"`},
		{"zip", "object::zip", "names, vals"},
		{"keep_keys", "object::keep_keys", `obj, "a", "b"`},
		{"merge", "object::merge", "left, right"},
		{"merge_recursive", "object::merge_deep", "left, right"},
		{"keys", "object::keys", "obj"},
		{"now", "datetime::now", ""},
		{"date", "datetime::parse", `"2026-09-15"`},
		{"date", "datetime::parse", `text, "2006-01-02"`},
		{"date_dayofweek", "datetime::day_of_week", "dt"},
		{"date_year", "datetime::year", "dt"},
		{"date_month", "datetime::month", "dt"},
		{"date_day", "datetime::day", "dt"},
		{"date_hour", "datetime::hour", "dt"},
		{"date_minute", "datetime::minute", "dt"},
		{"date_second", "datetime::second", "dt"},
		{"date_millisecond", "datetime::millisecond", "dt"},
		{"date_dayofyear", "datetime::day_of_year", "dt"},
		{"date_leapyear", "datetime::is_leap_year", "dt"},
		{"date_quarter", "datetime::quarter", "dt"},
		{"date_days_in_month", "datetime::days_in_month", "dt"},
		{"date_format", "datetime::format", "dt, layout"},
		{"date_add", "datetime::add", `dt, 1, "day"`},
		{"date_subtract", "datetime::subtract", `dt, 1, "day"`},
		{"pi", "math::pi", ""},
		{"abs", "math::abs", "value"},
		{"acos", "math::acos", "value"},
		{"asin", "math::asin", "value"},
		{"atan", "math::atan", "value"},
		{"atan2", "math::atan2", "y, x"},
		{"ceil", "math::ceil", "value"},
		{"cos", "math::cos", "value"},
		{"degrees", "math::degrees", "value"},
		{"exp", "math::exp", "value"},
		{"exp2", "math::exp2", "value"},
		{"floor", "math::floor", "value"},
		{"log", "math::log", "value"},
		{"log2", "math::log2", "value"},
		{"log10", "math::log10", "value"},
		{"pow", "math::pow", "value, 2"},
		{"radians", "math::radians", "value"},
		{"round", "math::round", "value"},
		{"sin", "math::sin", "value"},
		{"sqrt", "math::sqrt", "value"},
		{"tan", "math::tan", "value"},
	}

	for _, test := range tests {
		for _, spelling := range []string{test.legacy, strings.ToUpper(test.legacy), strings.ToUpper(test.legacy[:1]) + test.legacy[1:]} {
			t.Run(spelling+"/"+test.args, func(t *testing.T) {
				input := "return " + spelling + "(" + test.args + ")"
				want := "return " + test.canonical + "(" + test.args + ")"
				assertFQLStdlibMigration(t, input, want, 0)
			})
		}
	}
}

func TestMigrateFQLStdlibStructuralEdits(t *testing.T) {
	tests := []struct{ name, input, want string }{
		{
			name: "nested and canonical calls",
			input: `let obj = json_parse(body)
let digest = sha1(encoding::json_stringify(obj))
return abs(date_year(date("2026-09-15")) - 2026)`,
			want: `let obj = encoding::json_parse(body)
let digest = crypto::sha1(encoding::json_stringify(obj))
return math::abs(datetime::year(datetime::parse("2026-09-15")) - 2026)`,
		},
		{
			name: "unicode comments and non-call identifiers",
			input: `// json_parse("привет")
let json_parse = "sha1(abs(date_year()))"
/* merge_recursive and café */
return { json_parse, abs: "λ", digest: sha1("привет") }`,
			want: `// json_parse("привет")
let json_parse = "sha1(abs(date_year()))"
/* merge_recursive and café */
return {
    json_parse,
    abs: "λ",
    digest: crypto::sha1(
        "привет"
    )
}`,
		},
		{
			name: "unbraced final loop header and body",
			input: `let label = "日本語"
FOR item IN values(json_parse(body))
    RETURN abs(item)`,
			want: `let label = "日本語"
return for item in object::values(encoding::json_parse(body)) {
    return math::abs(item)
}`,
		},
		{
			name:  "braced final loop",
			input: "for item in values(obj) { return abs(item) }",
			want:  "return for item in object::values(obj) {\n    return math::abs(item)\n}",
		},
		{
			name:  "error operators",
			input: "return [json_parse(body)?, abs(-2)]",
			want:  "return [encoding::json_parse(body)?, math::abs(-2)]",
		},
		{
			name:  "eligible child of a qualified call",
			input: "return custom::abs(abs(-2))",
			want:  "return custom::abs(math::abs(-2))",
		},
		{
			name: "calls inside functions",
			input: `func calculate(x) { return abs(x) }
return calculate(-1)`,
			want: `func calculate(x) {
    return math::abs(x)
}
return calculate(-1)`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertFQLStdlibMigration(t, test.input, test.want, 0)
		})
	}
}

func TestMigrateFQLStdlibDeferredCalls(t *testing.T) {
	tests := []struct{ call, reason string }{
		{`join("a", "b")`, "legacy path::join"},
		{`join(["a", "b"], ",")`, "modern global string joining"},
		{"keys(obj, true)", "argument-aware"},
		{"keys(obj, false)", "argument-aware"},
		{"keys(obj, option)", "argument-aware"},
		{"keys()", "only keys(obj)"},
		{"keys(obj, true, extra)", "argument-aware"},
		{`date_compare(a, b, "year")`, "component-range"},
		{`date_compare(a, b, "year", "day")`, "not a drop-in"},
		{`date_diff(a, b, "day")`, "integer/floating"},
		{`date_diff(a, b, "day", true)`, "datetime::diff"},
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
			result := assertFQLStdlibMigration(t, input, input, 1)
			action := result.ManualActions[0]
			if action.Path != "query.fql" || action.Line != 2 || !strings.Contains(action.Reason, test.reason) {
				t.Fatalf("unexpected manual action: %#v", action)
			}
		})
	}
}

func TestMigrateFQLStdlibMixedManualActions(t *testing.T) {
	input := `let text = "λ"
FOR item IN values(obj)
    RETURN [keys(item, true), average([abs(-2)]), average(item)]`
	want := `let text = "λ"
return for item in object::values(obj) {
    return [keys(item, true), average([math::abs(-2)]), average(item)]
}`
	result := assertFQLStdlibMigration(t, input, want, 3)
	for _, action := range result.ManualActions {
		if action.Line != 3 || action.Path != "query.fql" {
			t.Fatalf("manual action lost its original source location: %#v", action)
		}
	}
}

func TestMigrateFQLStdlibLeavesUnrelatedSourceUnchanged(t *testing.T) {
	inputs := []string{
		`RETURN [encoding::json_parse(body), crypto::sha1(text), path::base(path), object::merge(a, b), datetime::year(dt), math::sqrt(x)]`,
		`RETURN [ENCODING::JSON_PARSE(body), custom::ABS(x), path::join("a", "b"), object::keys(obj)]`,
		`RETURN [union(a, b), union_distinct(a, b), nth(a, 0), minus(a, b), push(a, 1), pop(a), shift(a), unshift(a, 1), position(a, 1), remove_nth(a, 0), outersection(a, b)]`,
		`RETURN [rand(), rand(10, 1), range(1, 10), random::float(), arrays::at(xs, 0)]`,
		`let json_parse = "abs(1)" // json_parse(body)
RETURN { json_parse, abs: "date_diff()" }`,
	}

	for _, input := range inputs {
		assertFQLStdlibMigration(t, input, input, 0)
	}
}

func TestMigrateFQLStdlibNameCollisions(t *testing.T) {
	tests := []struct{ name, input, want, reason string }{
		{
			name: "local declaration",
			input: `func abs(x) { return x }
return abs(-1)`,
			reason: "file-local",
		},
		{
			name: "forward declaration and safe sibling",
			input: `let x = abs(-1)
func abs(x) { return x }
return sha1(x)`,
			want: `let x = abs(-1)
func abs(x) {
    return x
}
return crypto::sha1(x)`,
			reason: "file-local",
		},
		{
			name: "nested declaration conservatively guards whole file",
			input: `func outer() { func abs(x) { return x } return 1 }
return abs(-1)`,
			reason: "file-local",
		},
		{
			name: "case insensitive guard",
			input: `func ABS(x) { return x }
return Abs(-1)`,
			reason: "file-local",
		},
		{
			name: "explicit function alias",
			input: `use custom::calculate as abs
return ABS(-1)`,
			reason: "function alias",
		},
		{
			name: "destination namespace alias",
			input: `use custom as math
return abs(-1)`,
			reason: "redirect the replacement math::abs",
		},
		{
			name: "function alias redirects destination namespace",
			input: `use custom::calculate as math
return abs(-1)`,
			reason: "redirect the replacement math::abs",
		},
		{
			name: "conflicting aliases cannot hide redirection",
			input: `use custom as math
use math as math
return abs(-1)`,
			reason: "redirect",
		},
		{
			name: "collision precedes semantic deferral",
			input: `func average(xs) { return xs }
return average(xs)`,
			reason: "file-local",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			want := test.want
			if want == "" {
				want = test.input
			}

			result := assertFQLStdlibMigration(t, test.input, want, 1)
			if !strings.Contains(result.ManualActions[0].Reason, test.reason) {
				t.Fatalf("unexpected collision explanation: %#v", result.ManualActions)
			}
		})
	}
}

func TestMigrateFQLStdlibUnrelatedAliases(t *testing.T) {
	for _, head := range []string{
		"use custom as other",
		"use custom as abs",
		"use custom::calculate as other",
		"use math as math",
		"use custom as MATH", // Alias resolution is case-sensitive; generated math::abs is unaffected.
	} {
		assertFQLStdlibMigration(t, head+"\nreturn abs(-1)", head+"\nreturn math::abs(-1)", 0)
	}

	input := "use custom as math\nRETURN math::abs(-1)"
	assertFQLStdlibMigration(t, input, input, 0)
}

func TestFQLStdlibEditsPreserveArgumentSource(t *testing.T) {
	input := `return SHA1 /* call comment */ (
    JSON_STRINGIFY({ abs: "json_parse()", value: 1.00, text: "日本語" }) // argument comment
)?`
	want := `return crypto::sha1 /* call comment */ (
    encoding::json_stringify({ abs: "json_parse()", value: 1.00, text: "日本語" }) // argument comment
)?`
	src := source.New("query.fql", input)
	program, err := parseFQLSource(src)
	if err != nil {
		t.Fatal(err)
	}

	edits, actions, err := planFQLStdlib(src, program)
	if err != nil {
		t.Fatal(err)
	}

	got, err := applyFQLEdits(input, edits)
	if err != nil {
		t.Fatal(err)
	}

	if got != want || len(actions) != 0 {
		t.Fatalf("stdlib edits changed argument source:\n%s\nmanual=%#v", got, actions)
	}

	// The current formatter drops the comment between the call name and '('.
	// Until core preserves it, the complete migration must fail without output.
	result, err := migrateFQLSource(src)
	if err == nil || !strings.Contains(err.Error(), "did not preserve comments") || result.Data != nil || result.Changed {
		t.Fatalf("unsafe formatting was accepted: result=%#v err=%v", result, err)
	}
}

func TestMigrateFQLStdlibCanonicalCallsExecute(t *testing.T) {
	input := `return [json_parse("[1]"), sha1("abc"), base("/tmp/file"), has({ a: 1 }, "a"), date_year(date("2026-09-15T00:00:00Z")), abs(-3)]`
	result, err := migrateFQLSource(source.New("query.fql", input))
	if err != nil {
		t.Fatal(err)
	}

	engine, err := ferret.New(ferret.WithStdlib(stdlib.Full()))
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if err := engine.Close(); err != nil {
			t.Errorf("close engine: %v", err)
		}
	})

	output, err := engine.Run(context.Background(), source.New("query.fql", string(result.Data)))
	if err != nil {
		t.Fatal(err)
	}

	want := `[[1],"a9993e364706816aba3e25717850c26c9cd0d89d","file",true,2026,3]`
	if string(output.Content) != want {
		t.Fatalf("migrated behavior = %s, want %s", output.Content, want)
	}
}

func assertFQLStdlibMigration(t *testing.T, input, want string, manualCount int) fqlMigrationResult {
	t.Helper()

	result, err := migrateFQLSource(source.New("query.fql", input))
	if err != nil {
		t.Fatal(err)
	}

	if len(result.ManualActions) != manualCount {
		t.Fatalf("manual actions = %#v, want %d", result.ManualActions, manualCount)
	}

	if input == want {
		if result.Changed || result.Data != nil {
			t.Fatalf("unchanged source was rewritten: %s", result.Data)
		}
	} else if !result.Changed || string(result.Data) != want {
		t.Fatalf("unexpected migration:\nwant:\n%s\ngot:\n%s", want, result.Data)
	}

	second, err := migrateFQLSource(source.New("query.fql", want))
	if err != nil {
		t.Fatal(err)
	}

	if second.Changed || second.Data != nil || len(second.ManualActions) != manualCount {
		t.Fatalf("second pass was not idempotent: %#v", second)
	}

	return result
}
