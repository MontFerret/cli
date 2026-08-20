package migration

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/antlr4-go/antlr/v4"

	"github.com/MontFerret/ferret/v2/pkg/diagnostics"
	"github.com/MontFerret/ferret/v2/pkg/formatter"
	"github.com/MontFerret/ferret/v2/pkg/parser"
	parserd "github.com/MontFerret/ferret/v2/pkg/parser/diagnostics"
	"github.com/MontFerret/ferret/v2/pkg/parser/fql"
	"github.com/MontFerret/ferret/v2/pkg/source"
)

func planFQLSourceChanges(ctx context.Context, project *migrationProject) (*fqlSourcePlan, error) {
	result := &fqlSourcePlan{ScannedFiles: len(project.FQLFiles)}

	for _, filename := range project.FQLFiles {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		snapshot, err := snapshotMigrationFile(filename)
		if err != nil {
			return nil, err
		}

		relative, err := migrationRelativePath(project.Root, filename, "FQL")
		if err != nil {
			return nil, err
		}

		src := source.New(relative, string(snapshot.Data))
		migration, err := migrateFQLSource(src)
		if err != nil {
			result.ManualActions = append(result.ManualActions, fqlManualAction(relative, src, err))

			continue
		}

		if !migration.Changed {
			continue
		}

		result.Changes = append(result.Changes, plannedChange{
			change: Change{
				Path:         relative,
				Before:       snapshot.Data,
				After:        migration.Data,
				BeforeExists: true,
			},
			before: snapshot,
			mode:   snapshot.Mode,
		})
		result.MigratedFiles++
	}

	sortManualActions(result.ManualActions)

	return result, nil
}

func migrateFQLSource(src *source.Source) (fqlMigrationResult, error) {
	loop, err := finalTopLevelFQLFor(src)
	if err != nil {
		return fqlMigrationResult{}, err
	}

	if loop == nil {
		return fqlMigrationResult{}, nil
	}

	migrated, err := rewriteFinalFQLFor(src.Content(), loop)
	if err != nil {
		return fqlMigrationResult{}, err
	}

	formatted, err := formatMigratedFQLSource(source.New(src.Name(), migrated))
	if err != nil {
		return fqlMigrationResult{}, err
	}

	return fqlMigrationResult{Data: formatted, Changed: true}, nil
}

func finalTopLevelFQLFor(src *source.Source) (fql.IForExpressionContext, error) {
	if !utf8.ValidString(src.Content()) {
		return nil, fmt.Errorf("parse Ferret source: source is not valid UTF-8")
	}

	program, err := parseFQLSource(src)
	if err != nil {
		return nil, err
	}

	body := program.Body()
	if body == nil || body.BodyExpression() != nil {
		return nil, nil
	}

	statements := body.AllBodyStatement()
	if len(statements) == 0 {
		return nil, nil
	}

	loop := statements[len(statements)-1].ForExpression()
	if loop == nil || loop.GetStart() == nil {
		return nil, nil
	}

	return loop, nil
}

func rewriteFinalFQLFor(content string, loop fql.IForExpressionContext) (string, error) {
	start, ok := fqlByteOffset(content, loop.GetStart().GetStart())
	if !ok {
		return "", fmt.Errorf("locate final top-level FOR in Ferret source")
	}

	if loop.OpenBrace() != nil {
		return content[:start] + "return " + content[start:], nil
	}

	headerStop := -1
	if header := loop.ForExpressionSource(); header != nil && header.GetStop() != nil {
		headerStop = header.GetStop().GetStop()
	} else if header := loop.Expression(); header != nil && header.GetStop() != nil {
		headerStop = header.GetStop().GetStop()
	}

	if headerStop < 0 || loop.GetStop() == nil {
		return "", fmt.Errorf("locate final top-level FOR boundaries in Ferret source")
	}

	headerEnd, ok := fqlByteOffset(content, headerStop+1)
	if !ok {
		return "", fmt.Errorf("locate final top-level FOR header in Ferret source")
	}

	loopEnd, ok := fqlByteOffset(content, loop.GetStop().GetStop()+1)
	if !ok || headerEnd > loopEnd {
		return "", fmt.Errorf("locate final top-level FOR body in Ferret source")
	}

	return content[:start] + "return " + content[start:headerEnd] + " {" +
		content[headerEnd:loopEnd] + "\n}" + content[loopEnd:], nil
}

func parseFQLSource(src *source.Source) (program *fql.ProgramContext, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			program = nil
			err = fmt.Errorf("parse Ferret source: %v", recovered)
		}
	}()

	handler := parserd.NewErrorHandler(src, 10)
	history := parserd.NewTokenHistory(64)
	p := parser.New(src.Content(), func(stream antlr.TokenStream) antlr.TokenStream {
		return parserd.NewTrackingTokenStream(stream, history)
	})
	p.RemoveErrorListeners()
	p.AddErrorListener(parserd.NewErrorListener(src, handler, history))

	program = p.Program()
	if handler.HasErrors() {
		return nil, fmt.Errorf("parse Ferret source: %w", handler.Errors().First())
	}

	return program, nil
}

func formatMigratedFQLSource(src *source.Source) (data []byte, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			data = nil
			err = fmt.Errorf("format migrated Ferret source: %v", recovered)
		}
	}()

	// Alpha.46 token positions count runes while its formatter slices bytes. ASCII placeholders make
	// those positions byte-safe; the exact runes are restored before the formatted source is validated.
	protected, replacements := protectFQLNonASCII(src.Content())

	var output bytes.Buffer
	if err := formatter.New().Format(&output, source.New(src.Name(), protected)); err != nil {
		return nil, fmt.Errorf("format migrated Ferret source: %w", err)
	}

	restored, err := restoreFQLNonASCII(output.String(), replacements)
	if err != nil {
		return nil, fmt.Errorf("format migrated Ferret source: %w", err)
	}

	formatted := []byte(restored)
	if !utf8.Valid(formatted) || !slices.Equal(nonASCIISequence(src.Content()), nonASCIISequence(restored)) {
		return nil, fmt.Errorf("format migrated Ferret source: formatter did not preserve non-ASCII source text")
	}

	if _, err := parseFQLSource(source.New(src.Name(), string(formatted))); err != nil {
		return nil, fmt.Errorf("validate formatted Ferret source: %w", err)
	}

	return formatted, nil
}

type fqlNonASCIIReplacement struct {
	marker string
	value  rune
}

func protectFQLNonASCII(content string) (string, []fqlNonASCIIReplacement) {
	if utf8.RuneCountInString(content) == len(content) {
		return content, nil
	}

	prefix := "ferretmigrateunicodeplaceholder"
	for strings.Contains(content, prefix) {
		prefix += "x"
	}

	var protected strings.Builder
	protected.Grow(len(content))

	var replacements []fqlNonASCIIReplacement
	for _, r := range content {
		if r < utf8.RuneSelf {
			protected.WriteRune(r)
			continue
		}

		marker := prefix + strconv.Itoa(len(replacements)) + "x"
		protected.WriteString(marker)
		replacements = append(replacements, fqlNonASCIIReplacement{marker: marker, value: r})
	}

	return protected.String(), replacements
}

func restoreFQLNonASCII(content string, replacements []fqlNonASCIIReplacement) (string, error) {
	for _, replacement := range replacements {
		if strings.Count(content, replacement.marker) != 1 {
			return "", fmt.Errorf("formatter did not preserve non-ASCII placeholder")
		}

		content = strings.Replace(content, replacement.marker, string(replacement.value), 1)
	}

	return content, nil
}

func nonASCIISequence(content string) []rune {
	var result []rune
	for _, r := range content {
		if r >= utf8.RuneSelf {
			result = append(result, r)
		}
	}

	return result
}

func fqlManualAction(path string, src *source.Source, err error) ManualAction {
	detail, line, _ := fqlDiagnosticDetails(src, err)

	return ManualAction{
		Path:   path,
		Detail: detail,
		Reason: "Ferret source could not be migrated safely; file was not modified",
		Line:   line,
	}
}

func fqlDiagnosticDetails(src *source.Source, err error) (detail string, line, column int) {
	detail = err.Error()
	line = 1
	column = 1

	var diagnostic *diagnostics.Diagnostic
	if !errors.As(err, &diagnostic) {
		return detail, line, column
	}

	detail = diagnostic.Message
	for _, span := range diagnostic.Spans {
		if !span.Main {
			continue
		}

		byteSpan, ok := fqlByteSpan(src.Content(), span.Span)
		if !ok {
			return detail, line, column
		}

		spanLine, spanColumn := src.LocationAt(byteSpan)
		if spanLine > 0 {
			line = spanLine
		}

		if spanColumn > 0 {
			column = spanColumn
		}

		return detail, line, column
	}

	return detail, line, column
}

func fqlByteSpan(content string, span source.Span) (source.Span, bool) {
	start, ok := fqlByteOffset(content, span.Start)
	if !ok {
		return source.Span{}, false
	}

	end, ok := fqlByteOffset(content, span.End)
	if !ok {
		return source.Span{}, false
	}

	return source.Span{Start: start, End: end}, true
}

func fqlByteOffset(content string, runeOffset int) (int, bool) {
	if runeOffset < 0 {
		return 0, false
	}

	current := 0
	for byteOffset := range content {
		if current == runeOffset {
			return byteOffset, true
		}

		current++
	}

	if current == runeOffset {
		return len(content), true
	}

	return 0, false
}
