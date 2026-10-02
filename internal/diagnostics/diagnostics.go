// Package diagnostics presents native and portable execution failures.
package diagnostics

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/MontFerret/api"
	portable "github.com/MontFerret/api/diagnostics"
	native "github.com/MontFerret/ferret/v2/pkg/diagnostics"
	"github.com/MontFerret/ferret/v2/pkg/source"
	"github.com/MontFerret/wire/client"
	"github.com/MontFerret/wire/pkg/failure"
)

// Format renders structured diagnostics throughout joined and wrapped errors.
// Portable projections take precedence over their duplicate native causes;
// unrelated failures, including cleanup failures, remain visible.
func Format(err error) string {
	var out strings.Builder
	projected := make(map[string]bool)
	visit(err, func(current error) {
		for _, diagnostic := range portableAt(current) {
			projected[diagnosticKey(diagnostic)] = true
		}
	})
	printed := make(map[string]bool)
	render(&out, err, projected, printed)

	return strings.TrimRight(out.String(), "\n")
}

// Print writes a formatted execution failure to the selected diagnostic stream.
func Print(out io.Writer, err error) {
	if err != nil {
		fmt.Fprintln(out, Format(err))
	}
}

func portableAt(err error) portable.Diagnostics {
	switch value := err.(type) {
	case portable.Diagnostics:
		return value
	case *client.Error:
		if value != nil {
			return value.Diagnostics
		}
	case *failure.Failure:
		if value != nil {
			return value.Diagnostics
		}
	}

	return nil
}

func visit(err error, fn func(error)) {
	if err == nil {
		return
	}

	fn(err)

	switch value := err.(type) {
	case interface{ Unwrap() []error }:
		for _, child := range value.Unwrap() {
			visit(child, fn)
		}
	case interface{ Unwrap() error }:
		visit(value.Unwrap(), fn)
	}
}

func render(out io.Writer, err error, projected, printed map[string]bool) {
	if err == nil {
		return
	}

	if values := portableAt(err); len(values) > 0 {
		for _, value := range values {
			key := diagnosticKey(value)
			if !printed[key] {
				printed[key] = true
				formatPortable(out, presentation(value))
			}
		}

		return
	}

	if value, ok := err.(*native.Diagnostic); ok && value != nil {
		if projected[diagnosticKey(portableDiagnostic(value))] {
			render(out, value.Cause, projected, printed)
		} else if separateCause(value.Cause, projected) {
			projectedValue := *value
			projectedValue.Cause = nil
			native.FormatDiagnostic(out, &projectedValue, 0)
			render(out, value.Cause, projected, printed)
		} else {
			native.FormatDiagnostic(out, value, 0)
		}

		return
	}

	switch value := err.(type) {
	case interface{ Unwrap() []error }:
		for _, child := range value.Unwrap() {
			render(out, child, projected, printed)
		}
	case interface{ Unwrap() error }:
		structured := false
		visit(value.Unwrap(), func(child error) {
			if len(portableAt(child)) > 0 {
				structured = true
			}

			if _, ok := child.(*native.Diagnostic); ok {
				structured = true
			}
		})

		if structured {
			render(out, value.Unwrap(), projected, printed)
		} else {
			fmt.Fprintln(out, native.Format(err))
		}
	default:
		fmt.Fprintln(out, native.Format(err))
	}
}

// Native's formatter follows one diagnostic cause. Split structured branches
// when that would hide joined failures or repeat an already portable projection.
func separateCause(err error, projected map[string]bool) bool {
	separate := false
	visit(err, func(current error) {
		if _, ok := current.(interface{ Unwrap() []error }); ok || len(portableAt(current)) > 0 {
			separate = true
		}

		if value, ok := current.(*native.Diagnostic); ok && value != nil && projected[diagnosticKey(portableDiagnostic(value))] {
			separate = true
		}
	})

	return separate
}

func presentation(value portable.Diagnostic) *native.Diagnostic {
	result := &native.Diagnostic{
		Kind: native.Kind(value.Kind), Message: value.Message,
		Source: source.New(value.Source.Name, value.Source.Content),
		Hint:   value.Hint, Note: value.Note,
	}

	for _, annotation := range value.Annotations {
		result.Spans = append(result.Spans, native.ErrorSpan{
			Span:  source.Span{Start: annotation.Range.Span.Start, End: annotation.Range.Span.End},
			Label: annotation.Message, Main: annotation.Primary,
		})
	}

	return result
}

func portableDiagnostic(value *native.Diagnostic) portable.Diagnostic {
	result := portable.Diagnostic{
		Kind: portable.Kind(value.Kind), Message: value.Message,
		Source: api.NewSource(value.Source.Name(), value.Source.Content()),
		Hint:   value.Hint, Note: value.Note,
	}

	for _, span := range value.Spans {
		rangeAt := value.Source.RangeAt(span.Span)
		result.Annotations = append(result.Annotations, portable.Annotation{
			Range: api.Range{
				Location: api.Location{SourceName: rangeAt.SourceName, Position: api.Position{Line: rangeAt.Line, Column: rangeAt.Column}},
				Span:     api.Span{Start: span.Span.Start, End: span.Span.End},
			},
			Message: span.Label, Primary: span.Main,
		})
	}

	return result
}

func diagnosticKey(value portable.Diagnostic) string {
	if len(value.Annotations) == 0 {
		value.Annotations = nil
	}

	encoded, _ := json.Marshal(value)

	return string(encoded)
}
