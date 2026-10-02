package runtime

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync/atomic"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/MontFerret/api"
	"github.com/MontFerret/ferret/v2/uapi"
	"github.com/MontFerret/wire/client"
)

// OpenSource creates source-execution resources for one invocation or REPL.
// Its context bounds Wire construction; callers pass their execution contexts
// to the returned api.Runtime and settle those calls before closing the owner.
func OpenSource(ctx context.Context, opts Options) (*SourceResources, error) {
	opts = NormalizeOptions(opts)

	if err := ValidateOptions(opts); err != nil {
		return nil, err
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if IsBuiltinType(opts.Type) {
		native, err := newBuiltin(opts)
		if err != nil {
			return nil, err
		}

		rt := uapi.Wrap(native.engine, api.Version(EmbeddedVersion()))

		return &SourceResources{Runtime: rt, owners: []io.Closer{rt, native}}, nil
	}

	if IsWireType(opts.Type) {
		return openWire(ctx, opts)
	}

	legacy, err := New(opts)
	if err != nil {
		return nil, err
	}

	return &SourceResources{Legacy: legacy, owners: []io.Closer{legacy}}, nil
}

func openWire(ctx context.Context, opts Options) (*SourceResources, error) {
	address, err := wireAddress(opts.Endpoint)
	if err != nil {
		return nil, err
	}

	connectCtx, cancel := context.WithTimeout(ctx, opts.ConnectTimeout)
	defer cancel()
	var dialed atomic.Bool

	connection, err := grpc.NewClient("passthrough:///"+address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithNoProxy(),
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			// gRPC reconnects transports by default. One Wire invocation owns
			// one physical connection; transport loss requires a new command.
			if !dialed.CompareAndSwap(false, true) {
				return nil, fmt.Errorf("wire transport connection was lost; start a new command")
			}

			return new(net.Dialer).DialContext(ctx, "tcp4", address)
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("connect Wire runtime %s: %w", opts.Endpoint, err)
	}

	physical := &transport{connection: connection}
	// Bound stream creation too: Wire detaches the long-lived Connect stream
	// from the construction context, so a stalled HTTP/2 handshake must close
	// the physical transport to unblock construction.
	closed := make(chan struct{})
	stop := context.AfterFunc(connectCtx, func() {
		_ = physical.Close()
		close(closed)
	})
	rt, err := client.New(connectCtx, connection)
	if !stop() {
		<-closed
	}

	if ctxErr := connectCtx.Err(); ctxErr != nil {
		err = errors.Join(err, ctxErr)
	}

	resources := &SourceResources{Runtime: rt, owners: []io.Closer{physical}}
	if rt != nil {
		resources.owners = []io.Closer{rt, physical}
	}

	if err != nil {
		return nil, errors.Join(fmt.Errorf("connect Wire runtime %s: %w", opts.Endpoint, err), resources.Close())
	}

	resources.Disconnected = physical.watch(connection)

	return resources, nil
}
