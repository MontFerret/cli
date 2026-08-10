package github

import "context"

type fakeTokenProvider struct {
	token string
	err   error
	calls int
}

func (provider *fakeTokenProvider) Token(context.Context) (string, error) {
	provider.calls++

	return provider.token, provider.err
}
