package browser

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
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

func (b *LinuxBrowser) Close(ctx context.Context, pid uint64) error {
	if pid > 0 {
		if err := exec.Command("kill", fmt.Sprintf("%d", pid)).Run(); err != nil {
			return ErrProcNotFound
		}
	}

	cmdStr := strings.Join(b.opts.ToFlags(), " ")

	psOut, err := exec.Command("ps", "-o", "pid=", "-o", "command=").Output()

	if err != nil {
		return ErrProcNotFound
	}

	r := regexp.MustCompile(`(\d+)\s(.+)`)

	for _, pair := range r.FindAllStringSubmatch(string(psOut), -1) {
		cmd := strings.TrimSpace(pair[2])

		if strings.HasSuffix(cmd, cmdStr) {
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

func (b *LinuxBrowser) findBinaryPath() (string, error) {
	variants := []string{
		"google-chrome-stable",
		"google-chrome-beta",
		"google-chrome-unstable",
		"chromium-browser",
		"chromium-browser-beta",
		"chromium-browser-unstable",
	}

	var result string

	// Find an installed one
	for _, name := range variants {
		_, err := exec.Command("which", name).Output()

		if err != nil {
			continue
		}

		result = name

		if result != "" {
			break
		}
	}

	if result == "" {
		return "", ErrBinNotFound
	}

	return result, nil
}
