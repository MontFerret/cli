package wirehost

import (
	"net"
	"sync"
	"sync/atomic"
)

type listener struct {
	net.Listener
	accepted    atomic.Int64
	mu          sync.Mutex
	connections map[net.Conn]struct{}
	changed     chan struct{}
}

func (l *listener) Accept() (net.Conn, error) {
	connection, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}

	tracked := &trackedConnection{Conn: connection, listener: l}
	l.mu.Lock()
	if l.connections == nil {
		l.connections = make(map[net.Conn]struct{})
	}

	l.connections[tracked] = struct{}{}
	l.mu.Unlock()
	l.accepted.Add(1)

	return tracked, nil
}
