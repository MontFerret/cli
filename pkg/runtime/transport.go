package runtime

import (
	"context"
	"io"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

// transport retains one physical close shared by construction cancellation and cleanup.
type transport struct {
	connection io.Closer
	once       sync.Once
	err        error
	stopWatch  context.CancelFunc
	watchDone  chan struct{}
}

// watch observes the established transport without probing or making RPCs.
// A loss is terminal for the logical Wire runtime; this never reconnects it.
func (t *transport) watch(connection *grpc.ClientConn) <-chan struct{} {
	ctx, cancel := context.WithCancel(context.Background())
	t.stopWatch = cancel
	t.watchDone = make(chan struct{})
	lost := make(chan struct{})
	go func() {
		defer close(t.watchDone)

		for connection.GetState() == connectivity.Ready {
			if !connection.WaitForStateChange(ctx, connectivity.Ready) {
				return
			}
		}

		if ctx.Err() == nil {
			close(lost)
		}
	}()

	return lost
}

func (t *transport) Close() error {
	t.once.Do(func() {
		if t.stopWatch != nil {
			t.stopWatch()
			<-t.watchDone
		}

		t.err = t.connection.Close()
	})

	return t.err
}
