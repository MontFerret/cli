package output

import (
	"bytes"
	"errors"
)

// wireDeliveryWriter counts without retaining bytes. Content validation is
// enabled only in tests, outside ordinary timing and allocation profiles.
type wireDeliveryWriter struct {
	bytes    int
	expected []byte
}

func (w *wireDeliveryWriter) Write(data []byte) (int, error) {
	if w.expected != nil {
		if len(data) > len(w.expected)-w.bytes || !bytes.Equal(data, w.expected[w.bytes:w.bytes+len(data)]) {
			return 0, errors.New("delivered content differs")
		}
	}

	w.bytes += len(data)

	return len(data), nil
}
