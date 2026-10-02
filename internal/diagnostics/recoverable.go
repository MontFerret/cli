package diagnostics

import (
	portable "github.com/MontFerret/api/diagnostics"
	native "github.com/MontFerret/ferret/v2/pkg/diagnostics"
	"github.com/MontFerret/wire/client"
	"github.com/MontFerret/wire/pkg/failure"
)

// Recoverable reports query failures that leave the runtime usable. Joined
// transport or cleanup failures are terminal even when a query diagnostic is present.
func Recoverable(err error) bool {
	if err == nil {
		return false
	}

	switch value := err.(type) {
	case portable.Diagnostics:
		return len(value) > 0
	case *native.Diagnostic:
		return value != nil
	case *client.Error:
		return value != nil && queryCategory(value.Category)
	case *failure.Failure:
		return value != nil && queryCategory(value.Category)
	case interface{ Unwrap() []error }:
		children := value.Unwrap()
		found := false

		for _, child := range children {
			if child == nil {
				continue
			}

			if !Recoverable(child) {
				return false
			}

			found = true
		}

		return found
	case interface{ Unwrap() error }:
		return Recoverable(value.Unwrap())
	default:
		return false
	}
}

func queryCategory(category failure.Category) bool {
	return category == failure.CategoryCompilation || category == failure.CategoryExecution
}
