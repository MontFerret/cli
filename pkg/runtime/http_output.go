package runtime

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/MontFerret/api"
)

var httpOutputBuffers = sync.Pool{New: func() any { return new([32 * 1024]byte) }}

func newHTTPOutput(ctx context.Context, cancel context.CancelFunc, response *http.Response) api.Output {
	metadata := api.Metadata{ContentType: response.Header.Get("Content-Type"), LengthKnown: response.ContentLength >= 0}
	if metadata.LengthKnown {
		metadata.Length = response.ContentLength
	}

	var once sync.Once
	var closeErr error
	closeBody := func() error {
		once.Do(func() {
			cancel()
			if err := response.Body.Close(); err != nil {
				closeErr = fmt.Errorf("close response data: %w", err)
			}
		})

		return closeErr
	}

	return newManagedOutput(ctx, metadata, func(operation context.Context, consumer api.Consumer) error {
		buffer := httpOutputBuffers.Get().(*[32 * 1024]byte)
		defer httpOutputBuffers.Put(buffer)
		var total int64
		var emptyReads int
		for {
			if err := operation.Err(); err != nil {
				return err
			}

			n, readErr := response.Body.Read(buffer[:])
			var failure error
			if readErr != nil && !errors.Is(readErr, io.EOF) {
				failure = fmt.Errorf("read response data: %w", readErr)
			}

			if n > 0 {
				total += int64(n)
				emptyReads = 0
				if err := operation.Err(); err != nil {
					return joinOutputErrors(failure, err)
				}

				if err := consumer(operation, buffer[:n]); err != nil {
					return joinOutputErrors(err, failure)
				}
			}

			if failure != nil {
				return failure
			}

			if errors.Is(readErr, io.EOF) {
				if metadata.LengthKnown && total != metadata.Length {
					return fmt.Errorf("read response data: content length mismatch: %w", io.ErrUnexpectedEOF)
				}

				if total == 0 {
					return consumer(operation, buffer[:0])
				}

				return nil
			}

			if n == 0 {
				emptyReads++
				if emptyReads >= 100 {
					return fmt.Errorf("read response data: %w", io.ErrNoProgress)
				}
			}
		}
	}, func() { _ = closeBody() }, closeBody)
}
