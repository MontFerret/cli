package runtime

import (
	"errors"
	"fmt"

	"github.com/MontFerret/api"
)

type remoteSessionOptions struct {
	params map[string]any
}

var _ api.SessionOptions = (*remoteSessionOptions)(nil)

// Parameters are copied into a request-local map and serialized before HTTP I/O.
// Nested values retain encoding/json conversion semantics and are not retained
// by the runtime after the request has been serialized.
func (o *remoteSessionOptions) SetParam(name string, value any) error {
	if o.params == nil {
		o.params = make(map[string]any)
	}

	o.params[name] = value

	return nil
}

func (o *remoteSessionOptions) SetParams(params map[string]any) error {
	if params != nil && o.params == nil {
		o.params = make(map[string]any, len(params))
	}

	for name, value := range params {
		o.params[name] = value
	}

	return nil
}

func (o *remoteSessionOptions) SetOutputContentType(string) error {
	return fmt.Errorf("HTTP Worker runtime cannot select an output codec; configure the Worker host: %w", errors.ErrUnsupported)
}

func (o *remoteSessionOptions) SetFSRoot(string) error {
	return fmt.Errorf("HTTP Worker runtime cannot select a filesystem root; configure the Worker host: %w", errors.ErrUnsupported)
}

func newRemoteSessionOptions(setters []api.SessionOption) (*remoteSessionOptions, error) {
	opts := &remoteSessionOptions{}
	var failures []error
	for _, setter := range setters {
		if setter == nil {
			continue
		}

		if err := setter(opts); err != nil {
			failures = append(failures, err)
		}
	}

	return opts, errors.Join(failures...)
}
