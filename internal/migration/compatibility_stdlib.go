package migration

import (
	"fmt"

	"github.com/MontFerret/ferret/v2/pkg/parser/fql"
	"github.com/MontFerret/ferret/v2/pkg/source"
)

func checkFQLStdlib(src source.Source, program *fql.ProgramContext) ([]CompatibilityDiagnostic, error) {
	var diagnostics []CompatibilityDiagnostic
	for _, finding := range analyzeFQLStdlib(program) {
		name := finding.name
		spelling := name.GetText()

		span, ok := fqlByteSpan(src.Content(), source.Span{
			Start: name.GetStart().GetStart(),
			End:   name.GetStop().GetStop() + 1,
		})
		if !ok || span.End <= span.Start {
			return nil, fmt.Errorf("locate stdlib call %s in Ferret source", spelling)
		}

		position := src.PositionAt(span)
		if position.Line == 0 || position.Column == 0 {
			return nil, fmt.Errorf("resolve stdlib call %s location", spelling)
		}

		diagnostic := CompatibilityDiagnostic{
			Path:    src.Name(),
			Message: fmt.Sprintf("Legacy stdlib call `%s` should use `%s`.", spelling, finding.target),
			Help:    "Preview automatic replacements with `ferret migrate run --print`.",
			Line:    position.Line,
			Column:  position.Column,
			Kind:    CompatibilityDiagnosticIssue,
		}
		if finding.reason != "" {
			diagnostic.Message = fmt.Sprintf("Stdlib call `%s` needs manual review.", spelling)
			diagnostic.Help = finding.reason
		}

		diagnostics = append(diagnostics, diagnostic)
	}

	return diagnostics, nil
}
