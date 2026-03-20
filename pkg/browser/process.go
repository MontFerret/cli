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

// ProcessMatcher determines if a running process matches the target browser command.
type ProcessMatcher func(processCmd, targetCmd string) bool

// openProcess runs the browser binary with the given flags.
// If detach is true, starts in background and returns PID.
func openProcess(ctx context.Context, path string, flags []string) (uint64, bool, error) {
	return openProcessWithOpts(ctx, path, flags, true)
}

func openProcessWithOpts(ctx context.Context, path string, flags []string, detach bool) (uint64, bool, error) {
	cmd := exec.CommandContext(ctx, path, flags...)

	if detach {
		if err := cmd.Start(); err != nil {
			return 0, true, err
		}

		return uint64(cmd.Process.Pid), true, nil
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout

	return 0, false, cmd.Run()
}

// killPID kills a process by PID using the given kill command.
func killPID(pid uint64, killCmd string, killArgs ...string) error {
	args := append(killArgs, fmt.Sprintf("%d", pid))

	return exec.Command(killCmd, args...).Run()
}

// findProcessByPS searches for a matching process using ps output and returns its PID.
func findProcessByPS(ctx context.Context, targetCmd string, matcher ProcessMatcher) (uint64, error) {
	psOut, err := exec.Command("ps", "-o", "pid=", "-o", "command=").Output()

	if err != nil {
		return 0, ErrProcNotFound
	}

	r := regexp.MustCompile(`(\d+)\s(.+)`)

	for _, pair := range r.FindAllStringSubmatch(string(psOut), -1) {
		cmd := strings.TrimSpace(pair[2])

		if matcher(cmd, targetCmd) {
			p, err := strconv.ParseUint(pair[1], 10, 64)

			if err == nil {
				return p, nil
			}
		}
	}

	return 0, ErrProcNotFound
}

// closePosixProcess kills a process by PID and finds/kills any matching child processes.
func closePosixProcess(ctx context.Context, pid uint64, targetCmd string, matcher ProcessMatcher) error {
	if pid > 0 {
		if err := killPID(pid, "kill"); err != nil {
			return ErrProcNotFound
		}
	}

	foundPID, err := findProcessByPS(ctx, targetCmd, matcher)

	if err != nil {
		return err
	}

	return exec.CommandContext(ctx, "kill", fmt.Sprintf("%d", foundPID)).Run()
}
