package browser

import (
	"context"
	"errors"
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

	cmd := exec.Command(path, b.opts.ToFlags()...)

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
		return exec.Command("kill", fmt.Sprintf("%d", pid)).Run()
	}

	binaryPath, err := b.findBinaryPath()

	if err != nil {
		return err
	}

	cmdStr := fmt.Sprintf("%s %s", binaryPath, strings.Join(b.opts.ToFlags(), " "))

	psOut, err := exec.Command("ps", "-o", "pid=", "-o", "command=").Output()

	if err != nil {
		return err
	}

	r := regexp.MustCompile("(\\d+)\\s(.+)")

	for _, pair := range r.FindAllStringSubmatch(string(psOut), -1) {
		cmd := pair[2]

		if strings.HasPrefix(cmd, cmdStr) {
			p, err := strconv.ParseUint(pair[1], 10, 64)

			if err == nil {
				pid = p
			}
		}
	}

	if pid == 0 {
		return errors.New("running browser not found")
	}

	return exec.Command("kill", fmt.Sprintf("%d", pid)).Run()
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
		out, err := exec.Command("which", name).Output()

		if err != nil {
			return "", err
		}

		result = string(out)

		if result != "" {
			break
		}
	}

	if result == "" {
		return "", errors.New("no compatible browser was found")
	}

	return result, nil
}
