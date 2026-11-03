package wirehost

import "github.com/MontFerret/api"

type closeRuntime struct {
	api.Runtime
	close func() error
}

func (r *closeRuntime) Close() error {
	return r.close()
}
