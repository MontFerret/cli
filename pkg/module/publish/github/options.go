package github

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

const defaultAPIURL = "https://api.github.com"

type (
	Option func(*submitterOptions) error

	submitterOptions struct {
		apiURL       string
		httpClient   *http.Client
		pollInterval time.Duration
		token        TokenProvider
	}
)

// WithAPIURL overrides the GitHub API root for testing or compatible proxies.
func WithAPIURL(value string) Option {
	return func(options *submitterOptions) error {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("GitHub API URL must not be empty")
		}
		options.apiURL = value

		return nil
	}
}

// WithHTTPClient injects the transport used for GitHub requests.
func WithHTTPClient(value *http.Client) Option {
	return func(options *submitterOptions) error {
		if value == nil {
			return fmt.Errorf("GitHub HTTP client must not be nil")
		}
		options.httpClient = value

		return nil
	}
}

// WithPollInterval overrides the delay used while an asynchronous fork becomes ready.
func WithPollInterval(value time.Duration) Option {
	return func(options *submitterOptions) error {
		if value <= 0 {
			return fmt.Errorf("GitHub poll interval must be positive")
		}
		options.pollInterval = value

		return nil
	}
}

// WithTokenProvider injects the GitHub credential resolver.
func WithTokenProvider(value TokenProvider) Option {
	return func(options *submitterOptions) error {
		if value == nil {
			return fmt.Errorf("GitHub token provider must not be nil")
		}
		options.token = value

		return nil
	}
}

func defaultSubmitterOptions() submitterOptions {
	return submitterOptions{
		apiURL:       defaultAPIURL,
		httpClient:   http.DefaultClient,
		pollInterval: 500 * time.Millisecond,
		token:        NewTokenSource(),
	}
}
