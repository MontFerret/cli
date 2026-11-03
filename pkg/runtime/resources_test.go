package runtime

import (
	"errors"
	"io"
	"testing"
)

func TestResourcesCleanupOrderAndFailures(t *testing.T) {
	logicalErr, transportErr := errors.New("logical cleanup"), errors.New("transport cleanup")
	rt := &scriptedRuntime{closeErr: logicalErr}
	closed := 0
	resources := &Resources{Runtime: rt, owners: []io.Closer{rt, closeFunc(func() error {
		if rt.closed != 1 {
			t.Error("transport closed before runtime")
		}

		closed++

		return transportErr
	})}}
	for range 2 {
		err := resources.Close()
		if !errors.Is(err, logicalErr) || !errors.Is(err, transportErr) {
			t.Fatalf("lost cleanup failure: %v", err)
		}
	}

	if closed != 1 || rt.closed != 1 {
		t.Fatalf("transport=%d runtime=%d", closed, rt.closed)
	}
}
