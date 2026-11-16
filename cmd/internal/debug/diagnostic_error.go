package debug

import "github.com/MontFerret/cli/v2/internal/diagnostics"

// The binary prints the returned error once. Format it here so compilation
// diagnostics keep their source excerpts and joined cleanup errors.
type diagnosticError struct {
	cause error
}

func (e *diagnosticError) Error() string {
	return diagnostics.Format(e.cause)
}

func (e *diagnosticError) Unwrap() error {
	return e.cause
}
