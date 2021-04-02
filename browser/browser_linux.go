package browser

import "context"

type LinuxBrowser struct {
	opts Options
}

func New(opts Options) Browser {
	return *LinuxBrowser{opts}
}

func (b *LinuxBrowser) Open(ctx context.Context) error {
	panic("implement me")
}

func (b *LinuxBrowser) Close(ctx context.Context) error {
	panic("implement me")
}
