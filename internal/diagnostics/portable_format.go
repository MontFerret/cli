package diagnostics

import (
	"fmt"
	"io"
	"sort"

	native "github.com/MontFerret/ferret/v2/pkg/diagnostics"
)

func formatPortable(out io.Writer, diagnostic *native.Diagnostic) {
	primaries := 0
	for _, span := range diagnostic.Spans {
		if span.Main {
			primaries++
		}
	}

	if primaries <= 1 {
		native.FormatDiagnostic(out, diagnostic, 0)

		return
	}

	// Native's full formatter selects a single primary span. Keep all portable
	// primary markers and use its span renderer for collections with several.
	header := *diagnostic
	header.Spans = nil
	header.Note = ""
	header.Hint = ""
	native.FormatDiagnostic(out, &header, 0)
	spans := append([]native.ErrorSpan(nil), diagnostic.Spans...)
	sort.SliceStable(spans, func(i, j int) bool { return spans[i].Span.Start < spans[j].Span.Start })
	renderer := native.SpanRenderer{CaretChar: '^'}
	for _, primary := range []bool{false, true} {
		for _, span := range spans {
			if span.Main == primary {
				renderer.Render(out, diagnostic.Source, span.Span, span.Label)
			}
		}
	}

	if diagnostic.Note != "" {
		fmt.Fprintf(out, "Note: %s\n", diagnostic.Note)
	}

	if diagnostic.Hint != "" {
		fmt.Fprintf(out, "Hint: %s\n", diagnostic.Hint)
	}
}
