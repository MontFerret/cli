package run

import "io"

type failingCloseReader struct {
	io.Reader
	err error
}

func (r *failingCloseReader) Close() error {
	return r.err
}
