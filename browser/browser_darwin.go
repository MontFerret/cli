package browser

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
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

	cmd := exec.CommandContext(ctx, path, b.opts.ToFlags()...)

	if b.opts.Detach {
		if err := cmd.Start(); err != nil {
			return 0, err
		}

		return uint64(cmd.Process.Pid), nil
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout

	return 0, cmd.Run()
}

func (b *DarwinBrowser) Close(ctx context.Context, pid uint64) error {
	if pid > 0 {
		if err := exec.Command("kill", fmt.Sprintf("%d", pid)).Run(); err != nil {
			return ErrProcNotFound
		}
	}

	binaryPath, err := b.findBinaryPath()

	if err != nil {
		return err
	}

	cmdStr := fmt.Sprintf("%s %s", binaryPath, strings.Join(b.opts.ToFlags(), " "))

	psOut, err := exec.Command("ps", "-o", "pid=", "-o", "command=").Output()

	if err != nil {
		return ErrProcNotFound
	}

	r := regexp.MustCompile(`(\d+)\s(.+)`)

	for _, pair := range r.FindAllStringSubmatch(string(psOut), -1) {
		cmd := pair[2]

		if strings.HasPrefix(cmd, cmdStr) {
			p, err := strconv.ParseUint(pair[1], 10, 64)

			if err == nil {
				pid = p
				break
			}
		}
	}

	if pid == 0 {
		return ErrProcNotFound
	}

	return exec.CommandContext(ctx, "kill", fmt.Sprintf("%d", pid)).Run()
}

func (b *DarwinBrowser) findBinaryPath() (string, error) {
	variants := []string{
		"Google Chrome",
		"Google Chrome Canary",
		"Chromium",
		"Chromium Canary",
	}

	var result string

	// Find an installed one
	for _, name := range variants {
		dir := filepath.Join("/Applications", fmt.Sprintf("%s.app", name))
		stat, err := os.Stat(dir)

		if err == nil && stat.IsDir() {
			result = filepath.Join(dir, "Contents/MacOS", name)

			break
		}
	}

	if result == "" {
		return "", ErrBinNotFound
	}

	return result, nil
}
