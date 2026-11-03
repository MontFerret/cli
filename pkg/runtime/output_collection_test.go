package runtime

import (
	"testing"

	"github.com/MontFerret/api"
)

func collectOutput(t testing.TB, output api.Output, err error) (content *api.Content, resultErr error) {
	t.Helper()
	if err != nil {
		return nil, err
	}

	if output == nil {
		t.Fatal("missing output handle")
	}

	defer func() { resultErr = joinOutputErrors(resultErr, output.Close()) }()
	content, resultErr = output.Collect(t.Context())
	return content, resultErr
}
