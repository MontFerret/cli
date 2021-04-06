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

type WindowsBrowser struct {
	opts Options
}

func New(opts Options) Browser {
	return &WindowsBrowser{opts}
}

func (b *WindowsBrowser) Open(ctx context.Context) (uint64, error) {
	path, err := b.findBinaryPath()

	if err != nil {
		return 0, err
	}

	args := []string{
		"(",
		"Start-Process",
		"-FilePath", fmt.Sprintf("'%s'", path),
		"-ArgumentList", strings.Join(b.opts.ToFlags(), ","),
		"-PassThru",
		").ID",
	}

	cmd := exec.Command("powershell", args...)

	out, err := cmd.Output()

	if err != nil {
		return 0, err
	}

	pid, err := strconv.ParseUint(strings.TrimSpace(string(out)), 10, 64)

	if err != nil {
		return 0, err
	}

	if b.opts.Detach {
		return pid, nil
	}

	<-ctx.Done()

	return 0, b.Close(context.Background(), pid)
}

func (b *WindowsBrowser) Close(ctx context.Context, pid uint64) error {
	if pid > 0 {
		if err := exec.Command("taskkill", "-pid", fmt.Sprintf("%d", pid)).Run(); err != nil {
			return ErrProcNotFound
		}
	}

	path, err := b.findBinaryPath()

	if err != nil {
		return err
	}

	opts := strings.Join(b.opts.ToFlags(), " ")
	psOut, err := exec.Command("WMIC", "path", "win32_process", "get", "Caption,Processid,Commandline").Output()

	if err != nil {
		return ErrProcNotFound
	}

	r := regexp.MustCompile(`([A-Za-z.]+)\s+([A-Za-z-=0-9.":\\\s]+)\s(\d+)`)
	targetCmd := fmt.Sprintf("%s %s", path, opts)

	outArr := strings.Split(strings.TrimSpace(string(psOut)), "\n")

	for _, str := range outArr {
		matches := r.FindAllStringSubmatch(str, -1)

		if len(matches) == 0 {
			continue
		}

		groups := matches[0]

		cmd := strings.TrimSpace(groups[2])
		processId := strings.TrimSpace(groups[3])

		if cmd == "" {
			continue
		}

		cmd = strings.ReplaceAll(cmd, `"`, "")

		if strings.HasSuffix(cmd, targetCmd) {
			p, err := strconv.ParseUint(processId, 10, 64)

			if err == nil {
				pid = p

				break
			}
		}
	}

	if pid == 0 {
		return ErrProcNotFound
	}

	return exec.Command("taskkill", "/PID", fmt.Sprintf("%d", pid)).Run()
}

func (b *WindowsBrowser) findBinaryPath() (string, error) {
	variants := []string{
		"C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe",
		//"C:\\Program Files\\Chromimum\\Application\\chrome.exe",
		//"C:\\Users\\User\\AppData\\Local\\Google\\Chrome SxS\\Application\\chrome.exe",
	}

	var result string

	// Find an installed one
	for _, name := range variants {
		_, err := os.Stat(name)

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
