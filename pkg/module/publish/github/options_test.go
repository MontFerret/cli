package github

import "testing"

func TestDefaultSubmitterOptionsUseBoundedHTTPClient(t *testing.T) {
	options := defaultSubmitterOptions()
	if options.httpClient == nil {
		t.Fatal("default HTTP client is nil")
	}
	if options.httpClient.Timeout != defaultHTTPTimeout {
		t.Fatalf("default HTTP timeout = %v, want %v", options.httpClient.Timeout, defaultHTTPTimeout)
	}
}
