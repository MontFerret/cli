package browser

import (
	"context"
	"errors"
	"os"
	"os/exec"
)

type DarwinBrowser struct {
	opts Options
}

func New(opts Options) Browser {
	return &DarwinBrowser{opts}
}

func (b *DarwinBrowser) Open(ctx context.Context) error {
	variants := []string{
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		"/Applications/Google Chrome Canary.app/Contents/MacOS/Google Chrome Canary",
		"/Applications/Chromium.app/Contents/MacOS/Chromium",
		"/Applications/Chromium Canary.app/Contents/MacOS/Chromium Canary",
	}

	var path string

	// Find an installed one
	for _, v := range variants {
		_, err := os.Stat(v)

		if err == nil {
			path = v
			break
		}
	}

	if path == "" {
		return errors.New("no compatible browser was found")
	}

	err := exec.Command(path, b.opts.ToFlags()...).Start()

	if err != nil {
		return err
	}

	return Wait(ctx, b.opts)
}

func (b *DarwinBrowser) Close(ctx context.Context) error {
	panic("implement me")
}
