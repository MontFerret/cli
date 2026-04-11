package browser

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type DarwinBrowser struct {
	opts Options
}

func New(opts Options) Browser {
	return &DarwinBrowser{opts}
}

func (b *DarwinBrowser) Open(ctx context.Context) (uint64, error) {
	path, err := b.findBinaryPath()

	if err != nil {
		return 0, err
	}

	flags, err := b.opts.ToFlags()

	if err != nil {
		return 0, err
	}

	pid, detached, err := openProcess(ctx, path, flags, b.opts.Detach)

	if err != nil || !detached {
		return 0, err
	}

	return pid, nil
}

func (b *DarwinBrowser) Close(ctx context.Context, pid uint64) error {
	binaryPath, err := b.findBinaryPath()

	if err != nil {
		return err
	}

	flags, err := b.opts.ToFlags()

	if err != nil {
		return err
	}

	targetCmd := fmt.Sprintf("%s %s", binaryPath, strings.Join(flags, " "))

	return closePosixProcess(ctx, pid, targetCmd, strings.HasPrefix)
}

func (b *DarwinBrowser) findBinaryPath() (string, error) {
	variants := []string{
		"Google Chrome",
		"Google Chrome Canary",
		"Chromium",
		"Chromium Canary",
	}

	for _, name := range variants {
		dir := filepath.Join("/Applications", fmt.Sprintf("%s.app", name))
		stat, err := os.Stat(dir)

		if err == nil && stat.IsDir() {
			return filepath.Join(dir, "Contents/MacOS", name), nil
		}
	}

	return "", ErrBinNotFound
}
