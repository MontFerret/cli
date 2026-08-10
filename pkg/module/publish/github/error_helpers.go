package github

import "errors"

func stageError(stage Stage, err error) error {
	return &Error{Stage: stage, Err: err}
}

func hasStatus(err error, status int) bool {
	var target *apiError

	return errors.As(err, &target) && target.StatusCode == status
}
