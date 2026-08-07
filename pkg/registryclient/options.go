package registryclient

import "time"

const (
	defaultTimeout  = 15 * time.Second
	maxResponseSize = 4 << 20
)
