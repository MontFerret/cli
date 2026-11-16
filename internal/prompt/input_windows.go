package prompt

import (
	"context"
	"os"
	"runtime"
	"time"

	"golang.org/x/sys/windows"
)

var cancelSynchronousIO = windows.NewLazySystemDLL("kernel32.dll").NewProc("CancelSynchronousIo")

// Preserve os.File's console decoding and redirected-input behavior. A console
// read is synchronous and must be canceled on the thread that issued it.
func processInput(ctx context.Context, stdin *os.File) (readFunc, func() error, error) {
	input, err := stdin.SyscallConn()
	if err != nil {
		return nil, nil, err
	}

	read := func(b []byte) (int, error) {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		thread, err := windows.OpenThread(windows.THREAD_TERMINATE, false, windows.GetCurrentThreadId())
		if err != nil {
			return 0, err
		}

		defer windows.CloseHandle(thread)
		done, stopped := make(chan struct{}), make(chan struct{})
		stop := context.AfterFunc(ctx, func() {
			defer close(stopped)
			// Cancellation can arrive between admission and the kernel read.
			// Retry until that read settles, before releasing the thread.
			retry := time.NewTicker(time.Millisecond)
			defer retry.Stop()
			for {
				cancelSynchronousIO.Call(uintptr(thread))

				// Redirected stdin may use overlapped I/O on the same handle.
				_ = input.Control(func(fd uintptr) { _ = windows.CancelIoEx(windows.Handle(fd), nil) })

				select {
				case <-done:
					return
				case <-retry.C:
				}
			}
		})

		n, err := stdin.Read(b)
		close(done)
		if !stop() {
			<-stopped
		}

		return n, err
	}

	return read, func() error { return nil }, nil
}
