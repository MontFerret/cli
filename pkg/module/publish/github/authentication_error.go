package github

import (
	"context"
	"errors"
	"fmt"
)

type authenticationError struct {
	err error
}

func newAuthenticationError(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}

	var existing *authenticationError
	if errors.As(err, &existing) {
		return err
	}

	return &authenticationError{err: err}
}

func (e *authenticationError) Error() string {
	return fmt.Sprintf(
		"%v; set GH_TOKEN or GITHUB_TOKEN, or run %q",
		e.err,
		"gh auth login --hostname github.com",
	)
}

func (e *authenticationError) Unwrap() error {
	return e.err
}
