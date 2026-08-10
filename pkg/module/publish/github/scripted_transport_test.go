package github

import "net/http"

type scriptedTransport struct {
	handle func(*http.Request) (*http.Response, error)
}

func (transport *scriptedTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport.handle(request)
}
