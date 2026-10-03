package wirehost

import "net"

type trackedConnection struct {
	net.Conn
	listener *listener
}

func (c *trackedConnection) Close() error {
	err := c.Conn.Close()
	c.listener.mu.Lock()
	delete(c.listener.connections, c)

	if c.listener.changed != nil {
		close(c.listener.changed)
		c.listener.changed = nil
	}

	c.listener.mu.Unlock()

	return err
}
