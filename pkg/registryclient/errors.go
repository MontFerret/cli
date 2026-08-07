package registryclient

import "errors"

var (
	// ErrNotFound indicates that a requested registry module does not exist.
	ErrNotFound = errors.New("registry module not found")
	// ErrUnavailable indicates that the registry could not serve a usable response.
	ErrUnavailable = errors.New("registry unavailable")
	// ErrMalformed indicates that a registry document violates the supported protocol.
	ErrMalformed = errors.New("malformed registry response")
)
