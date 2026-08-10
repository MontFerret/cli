package github

import "fmt"

type apiError struct {
	StatusCode  int
	Message     string
	Permissions string
}

func (e *apiError) Error() string {
	message := fmt.Sprintf("GitHub API returned HTTP %d", e.StatusCode)
	if e.Message != "" {
		message += ": " + e.Message
	}

	if e.Permissions != "" {
		message += " (accepted permissions: " + e.Permissions + ")"
	}

	return message
}
