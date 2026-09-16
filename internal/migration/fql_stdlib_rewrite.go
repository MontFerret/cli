package migration

import (
	"fmt"

	"github.com/antlr4-go/antlr/v4"

	"github.com/MontFerret/ferret/v2/pkg/parser/fql"
	"github.com/MontFerret/ferret/v2/pkg/source"
)

// A rewrite changes only call boundaries and an optional trailing literal.
// Retained arguments remain in the original source, available for descendant edits.
type fqlStdlibRewrite struct {
	target     string
	wrapper    string
	dropLast   bool
	addLast    string
	namespaces []string
	reason     string
}

func (r fqlStdlibRewrite) description() string {
	if r.wrapper != "" {
		return r.wrapper + "(" + r.target + "(...))"
	}

	if r.addLast != "" {
		return r.target + "(..., " + r.addLast + ")"
	}

	return r.target
}

func (r fqlStdlibRewrite) edits(content string, call fql.IFunctionCallContext) ([]fqlSourceEdit, error) {
	var edits []fqlSourceEdit

	add := func(start, stop antlr.Token, text string) error {
		span, ok := fqlByteSpan(content, source.Span{Start: start.GetStart(), End: stop.GetStop() + 1})
		if !ok || span.End <= span.Start {
			return fmt.Errorf("locate stdlib call %s in Ferret source", call.FunctionName().GetText())
		}

		edits = append(edits, fqlSourceEdit{start: span.Start, end: span.End, text: text})

		return nil
	}

	target := r.target
	closing := ")"
	if r.wrapper != "" {
		target = r.wrapper + "(" + target
		closing += ")"
	}

	name := call.FunctionName()
	if err := add(name.GetStart(), name.GetStop(), target); err != nil {
		return nil, err
	}

	args := call.ArgumentList()

	if r.dropLast {
		expressions := args.AllExpression()
		comma := args.Comma(len(expressions) - 2).GetSymbol()
		if err := add(comma, comma, ""); err != nil {
			return nil, err
		}

		// Delete syntax tokens only: comments inside grouping parentheses or
		// between a unary minus and its integer must survive argument removal.
		last := expressions[len(expressions)-1]
		stream := call.GetParser().GetTokenStream()
		for i := last.GetStart().GetTokenIndex(); i <= last.GetStop().GetTokenIndex(); i++ {
			token := stream.Get(i)
			if token.GetChannel() != antlr.TokenDefaultChannel {
				continue
			}

			if err := add(token, token, ""); err != nil {
				return nil, err
			}
		}
	}

	if r.addLast != "" {
		separator := ", "
		if len(args.AllComma()) == len(args.AllExpression()) {
			separator = ""
		}

		closing = separator + r.addLast + closing
	}

	if closing != ")" {
		// Replace the delimiter instead of inserting after it: final-FOR edits
		// may already insert at the end of this call, including in loop headers.
		token := call.CloseParen().GetSymbol()
		if err := add(token, token, closing); err != nil {
			return nil, err
		}
	}

	return edits, nil
}
