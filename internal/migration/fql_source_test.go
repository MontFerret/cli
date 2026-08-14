package migration

import (
	"context"
	"strings"
	"testing"

	ferret "github.com/MontFerret/ferret/v2"
	"github.com/MontFerret/ferret/v2/pkg/source"
)

func TestMigrateFQLSourceFinalTopLevelFor(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name: "final loop",
			input: `FOR x IN 1..10
    RETURN x`,
			want: `return for x in 1..10 {
    return x
}`,
		},
		{
			name: "statements before final loop",
			input: `LET xs = 1..10
FOR x IN xs
    RETURN x`,
			want: `let xs = 1..10
return for x in xs {
    return x
}`,
		},
		{
			name: "final while loop",
			input: `VAR i = 0
FOR WHILE i < 3
    i += 1
    RETURN i`,
			want: `var i = 0
return for while i < 3 {
    i += 1
    return i
}`,
		},
		{
			name: "nested loop wraps outer only",
			input: `FOR x IN 1..10 {
    FOR y IN 1..10 {
        RETURN x * y
    }
}`,
			want: `return for x in 1..10 {
    for y in 1..10 {
        return x * y
    }
}`,
		},
		{
			name: "unicode before final loop uses byte offset",
			input: `LET label = "привет"
FOR x IN 1..3
    RETURN { label, x }`,
			want: `let label = "привет"
return for x in 1..3 {
    return { label, x }
}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := migrateFQLSource(source.New("query.fql", test.input))
			if err != nil {
				t.Fatal(err)
			}

			if !result.Changed {
				t.Fatal("expected source migration")
			}

			if got := string(result.Data); got != test.want {
				t.Fatalf("unexpected migration:\nwant:\n%s\ngot:\n%s", test.want, got)
			}
		})
	}
}

func TestMigrateFQLSourceLeavesOtherProgramsUnchanged(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name: "non-final loop",
			input: `FOR x IN 1..10 {
    PRINT(x)
}
RETURN 42`,
		},
		{
			name: "already explicit loop result",
			input: `RETURN FOR x IN 1..10 {
    RETURN x
}`,
		},
		{
			name: "assigned loop",
			input: `LET xs = (
    FOR x IN 1..10 {
        RETURN x
    }
)
RETURN xs`,
		},
		{
			name: "loop inside function",
			input: `FUNC values() {
    FOR x IN 1..10 {
        RETURN x
    }
}
RETURN values()`,
		},
		{
			name:  "effect-only script",
			input: "CLICK(button)\nWAITFOR EXISTS confirmation",
		},
		{
			name:  "explicit return none",
			input: "CLICK(button)\nRETURN NONE",
		},
		{
			name: "FOR in string and comment",
			input: `LET message = "FOR x IN xs RETURN x"
// FOR foo IN bar RETURN foo`,
		},
		{
			name:  "formatting alone is not migration",
			input: "LET value = 1\nRETURN value",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := migrateFQLSource(source.New("query.fql", test.input))
			if err != nil {
				t.Fatal(err)
			}

			if result.Changed || result.Data != nil {
				t.Fatalf("unexpected migration: %#v", result)
			}
		})
	}
}

func TestMigrateFQLSourcePreservesComments(t *testing.T) {
	input := `// before final loop
FOR x IN 1..3
    // inside loop
    RETURN x
// after final loop`

	result, err := migrateFQLSource(source.New("comments.fql", input))
	if err != nil {
		t.Fatal(err)
	}

	if !result.Changed {
		t.Fatal("expected source migration")
	}

	got := string(result.Data)
	for _, comment := range []string{"// before final loop", "// inside loop", "// after final loop"} {
		if !strings.Contains(got, comment) {
			t.Fatalf("migration dropped %q:\n%s", comment, got)
		}
	}

	if !strings.Contains(got, "return for x in 1..3") {
		t.Fatalf("final loop was not returned:\n%s", got)
	}
}

func TestMigrateFQLSourceReportsInvalidSource(t *testing.T) {
	src := source.New("broken.fql", "LET ok = 1\nFOR x IN\n    RETURN x")
	result, err := migrateFQLSource(src)
	if err == nil {
		t.Fatal("expected parse error")
	}

	if result.Changed || result.Data != nil {
		t.Fatalf("invalid source was changed: %#v", result)
	}

	action := fqlManualAction("broken.fql", src, err)
	if action.Path != "broken.fql" || action.Line < 2 || action.Detail == "" ||
		action.Reason != "Ferret source could not be migrated safely; file was not modified" {
		t.Fatalf("unexpected manual action: %#v", action)
	}
}

func TestMigrateFQLSourceReportsInvalidUTF8(t *testing.T) {
	result, err := migrateFQLSource(source.New("broken.fql", string([]byte{'F', 'O', 'R', ' ', 0xff})))
	if err == nil || !strings.Contains(err.Error(), "not valid UTF-8") {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Changed || result.Data != nil {
		t.Fatalf("invalid UTF-8 was changed: %#v", result)
	}
}

func TestMigrateFQLSourceExecutesAndIsIdempotent(t *testing.T) {
	first, err := migrateFQLSource(source.New("query.fql", "FOR x IN 1..3\n    RETURN x"))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := parseFQLSource(source.New("query.fql", string(first.Data))); err != nil {
		t.Fatalf("migrated source does not parse: %v", err)
	}

	engine, err := ferret.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := engine.Close(); err != nil {
			t.Errorf("close engine: %v", err)
		}
	})

	output, err := engine.Run(context.Background(), source.New("query.fql", string(first.Data)))
	if err != nil {
		t.Fatal(err)
	}

	if got, want := string(output.Content), "[1,2,3]"; got != want {
		t.Fatalf("migrated result = %s, want %s", got, want)
	}

	second, err := migrateFQLSource(source.New("query.fql", string(first.Data)))
	if err != nil {
		t.Fatal(err)
	}

	if second.Changed || second.Data != nil {
		t.Fatalf("second migration changed source: %#v", second)
	}
}
