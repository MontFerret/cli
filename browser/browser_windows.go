package browser

import "context"

type WindowsBrowser struct {
	opts Options
}

func New(opts Options) Browser {
	return &WindowsBrowser{opts}
}

func (b *WindowsBrowser) Open(ctx context.Context) error {
	panic("implement me")
}

func (b *WindowsBrowser) Close(ctx context.Context) error {
	panic("implement me")
}
