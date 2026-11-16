//go:build unix

package prompt

import (
	"context"
	"errors"
	"io"
	"os"

	"golang.org/x/sys/unix"
)

// Poll the original stdin (terminal, pipe, or file) alongside a private wake
// pipe. Closing the write end wakes poll without consuming any input.
func processInput(ctx context.Context, stdin *os.File) (readFunc, func() error, error) {
	input, err := stdin.SyscallConn()
	if err != nil {
		return nil, nil, err
	}

	wake, signal, err := os.Pipe()
	if err != nil {
		return nil, nil, err
	}

	closed := make(chan struct{})
	var closeErr error
	stop := context.AfterFunc(ctx, func() {
		closeErr = signal.Close()
		close(closed)
	})

	fds := []unix.PollFd{
		{Events: unix.POLLIN},
		{Fd: int32(wake.Fd()), Events: unix.POLLIN},
	}
	read := func(b []byte) (int, error) {
		var pollErr error
		err := input.Control(func(fd uintptr) {
			fds[0].Fd = int32(fd)
			for {
				_, pollErr = unix.Poll(fds, -1)
				if !errors.Is(pollErr, unix.EINTR) {
					break
				}
			}
		})
		if err != nil || pollErr != nil {
			return 0, errors.Join(err, pollErr)
		}

		if fds[1].Revents != 0 || ctx.Err() != nil {
			return 0, io.EOF
		}

		return stdin.Read(b)
	}

	return read, func() error {
		if stop() {
			closeErr = signal.Close()
		} else {
			<-closed
		}

		return errors.Join(closeErr, wake.Close())
	}, nil
}
