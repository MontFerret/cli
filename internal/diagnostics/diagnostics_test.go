package diagnostics

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/MontFerret/api"
	portable "github.com/MontFerret/api/diagnostics"
	native "github.com/MontFerret/ferret/v2/pkg/diagnostics"
	"github.com/MontFerret/wire/client"
	"github.com/MontFerret/wire/pkg/failure"
)

func TestPortablePresentation(t *testing.T) {
	diagnostic := portable.Diagnostic{
		Source: api.NewSource("query.fql", "RETURN"), Kind: "SyntaxError", Message: "Missing value",
		Annotations: []portable.Annotation{{Range: api.Range{Location: api.Location{SourceName: "query.fql", Position: api.Position{Line: 1, Column: 7}}, Span: api.Span{Start: 6, End: 6}}, Message: "missing value", Primary: true}},
		Hint:        "Provide a value", Note: "Produced by host",
	}
	values := portable.Diagnostics{diagnostic}
	cleanup := errors.New("close physical connection failed")
	for _, err := range []error{values, &client.Error{Category: failure.CategoryCompilation, Diagnostics: values}, &failure.Failure{Category: failure.CategoryCompilation, Diagnostics: values}} {
		text := Format(errors.Join(fmt.Errorf("execute: %w", err), cleanup))
		want := "SyntaxError: Missing value\n --> query.fql:1:7\n  |\n1 | RETURN\n  |       ^ missing value\nNote: Produced by host\nHint: Provide a value\nclose physical connection failed"
		if text != want {
			t.Fatalf("presentation = %q, want %q", text, want)
		}
	}

	text := Format(errors.Join(values, presentation(diagnostic)))
	if strings.Count(text, "SyntaxError: Missing value") != 1 {
		t.Fatalf("duplicate projection:\n%s", text)
	}

	if !Recoverable(values) || Recoverable(errors.Join(values, cleanup)) || Recoverable(&client.Error{Category: failure.CategoryConnectionNotFound}) {
		t.Fatal("incorrect recoverable classification")
	}
}

func TestPortableAnnotationsAndInvalidCoordinates(t *testing.T) {
	value := portable.Diagnostic{
		Source: api.NewSource("query.fql", "RETURN )"), Kind: "CustomHostKind", Message: "Unexpected token",
		Annotations: []portable.Annotation{
			{Range: api.Range{Span: api.Span{Start: 0, End: 6}}, Message: "statement"},
			{Range: api.Range{Span: api.Span{Start: 7, End: 8}}, Message: "offending token", Primary: true},
		},
	}
	text := Format(portable.Diagnostics{value})
	for _, want := range []string{"CustomHostKind", "^^^^^^ statement", "       ^ offending token", "query.fql:1:8"} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q:\n%s", want, text)
		}
	}

	value.Annotations[1].Range.Span.End = 9
	adapted := presentation(value)
	if adapted.Cause != nil || adapted.Spans[1].Span.End != 9 || adapted.Spans[1].Main != true {
		t.Fatal("coordinates or primary marker changed")
	}

	if strings.Contains(Format(portable.Diagnostics{value}), "offending token") {
		t.Fatal("invalid coordinate was clamped")
	}
}

func TestNativePresentationUnchanged(t *testing.T) {
	err := &native.Diagnostic{Kind: native.UnexpectedError, Message: "native failure"}
	if Format(err) != strings.TrimSpace(native.Format(err)) {
		t.Fatal("native rendering changed")
	}
}

func TestJoinedNativeCausesRetainCleanupWithoutDuplicateProjection(t *testing.T) {
	outer := &native.Diagnostic{Kind: native.UnexpectedError, Message: "outer failure"}
	inner := &native.Diagnostic{Kind: native.UnexpectedError, Message: "inner failure"}
	cleanup := errors.New("cleanup failed")
	outer.Cause = errors.Join(inner, cleanup)
	projections := portable.Diagnostics{portableDiagnostic(outer), portableDiagnostic(inner)}

	for _, err := range []error{outer, errors.Join(projections, outer)} {
		text := Format(err)
		for _, want := range []string{"outer failure", "inner failure", "cleanup failed"} {
			if strings.Count(text, want) != 1 {
				t.Fatalf("lost or duplicated %q:\n%s", want, text)
			}
		}
	}
}

func TestPortableMultiplePrimaryAnnotations(t *testing.T) {
	value := portable.Diagnostic{
		Source: api.NewSource("query.fql", "RETURN )"), Kind: "CustomHostKind", Message: "Two related ranges",
		Annotations: []portable.Annotation{
			{Range: api.Range{Span: api.Span{Start: 0, End: 6}}, Message: "first primary", Primary: true},
			{Range: api.Range{Span: api.Span{Start: 7, End: 8}}, Message: "second primary", Primary: true},
		},
		Note: "Both ranges matter", Hint: "Correct both",
	}
	text := Format(portable.Diagnostics{value})
	want := "CustomHostKind: Two related ranges\n --> query.fql:1:1\n  |\n1 | RETURN )\n  | ^^^^^^ first primary\n --> query.fql:1:8\n  |\n1 | RETURN )\n  |        ^ second primary\nNote: Both ranges matter\nHint: Correct both"
	if text != want {
		t.Fatalf("presentation = %q, want %q", text, want)
	}

	for _, span := range presentation(value).Spans {
		if !span.Main {
			t.Fatal("primary marker changed")
		}
	}
}
