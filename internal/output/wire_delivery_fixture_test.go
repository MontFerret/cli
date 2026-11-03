package output

import (
	"context"

	"github.com/MontFerret/api"
)

// wireDeliveryFixture borrows immutable payload storage. The host owns the
// only consumption; Close has no external resource or retained history.
type wireDeliveryFixture struct {
	payload       []byte
	callbackBytes int
}

func (o *wireDeliveryFixture) Metadata() api.Metadata {
	return api.Metadata{ContentType: "application/octet-stream", Length: int64(len(o.payload)), LengthKnown: true}
}

func (o *wireDeliveryFixture) Consume(ctx context.Context, consumer api.Consumer) error {
	for offset := 0; offset < len(o.payload); offset += o.callbackBytes {
		if err := ctx.Err(); err != nil {
			return err
		}

		if err := consumer(ctx, o.payload[offset:min(offset+o.callbackBytes, len(o.payload))]); err != nil {
			return err
		}
	}

	return nil
}

func (o *wireDeliveryFixture) Collect(context.Context) (*api.Content, error) {
	panic("delivery fixture must not be materialized")
}

func (o *wireDeliveryFixture) Close() error { return nil }
