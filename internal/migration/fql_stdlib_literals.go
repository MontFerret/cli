package migration

import (
	"strconv"
	"strings"

	"github.com/MontFerret/ferret/v2/pkg/parser/fql"
)

func fqlCallArguments(call fql.IFunctionCallContext) []fql.IExpressionContext {
	if args := call.ArgumentList(); args != nil {
		return args.AllExpression()
	}

	return nil
}

// Only descend through grouping and the grammar's single-child expression
// layers. Operators, recovery tails, and other expressions are never evaluated.
func fqlLiteralAtom(expression fql.IExpressionContext) fql.IExpressionAtomContext {
	for expression != nil && expression.GetChildCount() == 1 {
		predicate := expression.Predicate()
		if predicate == nil || predicate.GetChildCount() != 1 {
			return nil
		}

		atom := predicate.ExpressionAtom()
		if atom == nil {
			return nil
		}

		if atom.OpenParen() != nil && atom.GetChildCount() == 3 && atom.Expression() != nil {
			expression = atom.Expression()

			continue
		}

		if atom.GetChildCount() == 1 && atom.Literal() != nil {
			return atom
		}

		return nil
	}

	return nil
}

func fqlBooleanLiteral(expression fql.IExpressionContext) (bool, bool) {
	atom := fqlLiteralAtom(expression)
	if atom == nil || atom.Literal().BooleanLiteral() == nil {
		return false, false
	}

	return strings.EqualFold(atom.Literal().BooleanLiteral().GetText(), "true"), true
}

func fqlNegativeIntegerLiteral(expression fql.IExpressionContext) bool {
	// Parentheses may surround the entire signed literal as well as its operand.
	for expression != nil && expression.GetChildCount() == 1 {
		predicate := expression.Predicate()
		if predicate == nil || predicate.GetChildCount() != 1 {
			return false
		}

		atom := predicate.ExpressionAtom()
		if atom == nil || atom.OpenParen() == nil || atom.GetChildCount() != 3 || atom.Expression() == nil {
			return false
		}

		expression = atom.Expression()
	}

	if expression == nil || expression.UnaryOperator() == nil || expression.UnaryOperator().Minus() == nil {
		return false
	}

	atom := fqlLiteralAtom(expression.GetRight())
	if atom == nil || atom.Literal().IntegerLiteral() == nil {
		return false
	}

	// Core compiles the unsigned operand with Atoi before applying unary minus.
	// Reject overflow rather than removing a literal that would fail compilation.
	value, err := strconv.Atoi(atom.Literal().IntegerLiteral().GetText())

	return err == nil && value > 0
}
