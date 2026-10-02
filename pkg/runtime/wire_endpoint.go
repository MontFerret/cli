package runtime

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// DefaultConnectTimeout bounds Wire transport establishment and its logical handshake.
const DefaultConnectTimeout = 5 * time.Second

// IsWireType identifies the explicit Wire source-execution mode.
func IsWireType(name string) bool {
	return normalizeRuntimeType(name) == "wire"
}

func wireAddress(endpoint string) (string, error) {
	const prefix = "tcp://127.0.0.1:"
	if endpoint == "" {
		return "", fmt.Errorf("wire runtime requires --runtime-endpoint tcp://127.0.0.1:<port>")
	}

	invalid := func() (string, error) {
		return "", fmt.Errorf("invalid Wire endpoint %q: use tcp://127.0.0.1:<port> with a port from 1 to 65535", endpoint)
	}

	if !strings.HasPrefix(endpoint, prefix) {
		return invalid()
	}

	port := strings.TrimPrefix(endpoint, prefix)
	if port == "" {
		return invalid()
	}

	for _, char := range port {
		if char < '0' || char > '9' {
			return invalid()
		}
	}

	number, err := strconv.ParseUint(port, 10, 16)
	if err != nil || number == 0 {
		return invalid()
	}

	return net.JoinHostPort("127.0.0.1", strconv.FormatUint(number, 10)), nil
}
