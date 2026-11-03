// Package encoded supplies scripted output for consumer tests.
package encoded

import (
	"context"
	"errors"

	"github.com/MontFerret/api"
)

// Output scripts encoded chunks and terminal failures without a runtime.
type Output struct {
	Chunks     [][]byte
	Err        error
	CloseErr   error
	Closed     int
	Descriptor api.Metadata
}

func (o *Output) Metadata() api.Metadata { return o.Descriptor }

func (o *Output) Consume(ctx context.Context, consumer api.Consumer) error {
	for _, chunk := range o.Chunks {
		if err := consumer(ctx, chunk); err != nil {
			return errors.Join(err, o.Err)
		}
	}

	return o.Err
}

func (o *Output) Collect(ctx context.Context) (*api.Content, error) {
	var content *api.Content
	err := o.Consume(ctx, func(_ context.Context, chunk []byte) error {
		if content == nil {
			content = &api.Content{Metadata: o.Descriptor}
		}

		content.Data = append(content.Data, chunk...)

		return nil
	})

	return content, err
}

func (o *Output) Close() error {
	o.Closed++

	return o.CloseErr
}
