package browser

import (
	"context"
	"os/exec"
	"strings"
)

type LinuxBrowser struct {
	opts Options
}

func New(opts Options) Browser {
	return &LinuxBrowser{opts}
}

func (b *LinuxBrowser) Open(ctx context.Context) (uint64, error) {
	path, err := b.findBinaryPath()

	if err != nil {
		return 0, err
	}

	pid, detached, err := openProcess(ctx, path, b.opts.ToFlags(), b.opts.Detach)

	if err != nil || !detached {
		return 0, err
	}

	return pid, nil
}

func (b *LinuxBrowser) Close(ctx context.Context, pid uint64) error {
	targetCmd := strings.Join(b.opts.ToFlags(), " ")

	return closePosixProcess(ctx, pid, targetCmd, strings.HasSuffix)
}

func (b *LinuxBrowser) findBinaryPath() (string, error) {
	variants := []string{
		"google-chrome-stable",
		"google-chrome-beta",
		"google-chrome-unstable",
		"chromium-browser",
		"chromium-browser-beta",
		"chromium-browser-unstable",
	}

	for _, name := range variants {
		if _, err := exec.Command("which", name).Output(); err == nil {
			return name, nil
		}
	}

	return "", ErrBinNotFound
}
