package browser

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

type DarwinBrowser struct {
	opts Options
}

func New(opts Options) Browser {
	return &DarwinBrowser{opts}
}

func (b *DarwinBrowser) Open(_ context.Context) error {
	path, err := b.findBinaryPath()

	if err != nil {
		return err
	}

	cmd := exec.Command(path, b.opts.ToFlags()...)

	if b.opts.Detach {
		if err := cmd.Start(); err != nil {
			return err
		}

		fmt.Println(cmd.Process.Pid)

		return nil
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout

	return cmd.Run()
}

func (b *DarwinBrowser) Close(_ context.Context, pid string) error {
	if pid != "" {
		return exec.Command("kill", pid).Run()
	}

	binaryPath, err := b.findBinaryPath()

	if err != nil {
		return err
	}

	cmdStr := fmt.Sprintf("%s %s", binaryPath, strings.Join(b.opts.ToFlags(), " "))
	//
	//println(strings.Join(flags, " "))

	psOut, err := exec.Command("ps", "-o", "pid=", "-o", "command=").Output()

	if err != nil {
		return err
	}

	r := regexp.MustCompile("(\\d+)\\s(.+)")

	for _, pair := range r.FindAllStringSubmatch(string(psOut), -1) {
		cmd := pair[2]

		if strings.HasPrefix(cmd, cmdStr) {
			pid = pair[1]
		}
	}

	if pid == "" {
		return errors.New("running browser not found")
	}

	return exec.Command("kill", pid).Run()
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
		return "", errors.New("no compatible browser was found")
	}

	return result, nil
}
