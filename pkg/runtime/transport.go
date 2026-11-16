package runtime

import (
	"io"
	"sync"
)

// transport retains one physical close shared by construction cancellation and cleanup.
type transport struct {
	connection io.Closer
	once       sync.Once
	err        error
}

func (t *transport) Close() error {
	t.once.Do(func() {
		t.err = t.connection.Close()
	})

	return t.err
}
