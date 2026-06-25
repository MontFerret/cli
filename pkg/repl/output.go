package repl

import (
	"errors"
	"io"
)

func writeResult(dst io.Writer, src io.ReadCloser) (err error) {
	defer func() {
		if closeErr := src.Close(); closeErr != nil {
			if err == nil {
				err = closeErr
			} else {
				err = errors.Join(err, closeErr)
			}
		}
	}()

	var buf [32 * 1024]byte
	var wrote bool
	var last byte

	for {
		n, readErr := src.Read(buf[:])

		if n > 0 {
			if err := writeAll(dst, buf[:n]); err != nil {
				return err
			}

			wrote = true
			last = buf[n-1]
		}

		if readErr == nil {
			continue
		}

		if errors.Is(readErr, io.EOF) {
			break
		}

		return readErr
	}

	if wrote && last != '\n' {
		return writeAll(dst, []byte("\n"))
	}

	return nil
}

// writeAll writes the entire data slice to the given writer.
// It returns io.ErrShortWrite if dst.Write returns n=0 without an error,
// indicating that the writer is not making any progress.
func writeAll(dst io.Writer, data []byte) error {
	for len(data) > 0 {
		n, err := dst.Write(data)

		if err != nil {
			return err
		}

		if n <= 0 {
			return io.ErrShortWrite
		}

		data = data[n:]
	}

	return nil
}
