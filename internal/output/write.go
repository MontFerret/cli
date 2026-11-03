// Package output writes encoded output without materializing the payload.
package output

import (
	"context"
	"errors"
	"io"

	"github.com/MontFerret/api"
)

// Write settles src and returns the complete result plus any writer or cleanup
// failures that must end an interactive shell. It borrows dst.
func Write(ctx context.Context, dst io.Writer, src api.Output, newline bool) (resultErr, failure error) {
	if src == nil {
		return nil, nil
	}

	var wrote bool
	var last byte
	resultErr = src.Consume(ctx, func(_ context.Context, data []byte) error {
		if err := writeAll(dst, data); err != nil {
			failure = err

			return err
		}

		if len(data) > 0 {
			wrote, last = true, data[len(data)-1]
		}

		return nil
	})
	if failure != nil && !errors.Is(resultErr, failure) {
		resultErr = errors.Join(resultErr, failure)
	}

	if newline && failure == nil && wrote && last != '\n' {
		failure = writeAll(dst, []byte("\n"))
		resultErr = errors.Join(resultErr, failure)
	}

	closeErr := src.Close()
	if closeErr != nil && !errors.Is(resultErr, closeErr) {
		resultErr = errors.Join(resultErr, closeErr)
	}

	failure = errors.Join(failure, closeErr)

	return resultErr, failure
}

func writeAll(dst io.Writer, data []byte) error {
	for len(data) > 0 {
		n, err := dst.Write(data)
		if err != nil {
			return err
		}

		if n <= 0 || n > len(data) {
			return io.ErrShortWrite
		}

		data = data[n:]
	}

	return nil
}
