package runtime

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestLegacyVersionClosesResponseAndRetainsFailure(t *testing.T) {
	closeErr := errors.New("close version response failed")
	for _, content := range []string{`{"version":{"ferret":"legacy"}}`, "invalid JSON"} {
		closed := 0
		body := &struct {
			io.Reader
			io.Closer
		}{strings.NewReader(content), closeFunc(func() error {
			closed++

			return closeErr
		})}
		rt := &Remote{url: url.URL{Scheme: "http", Host: "worker.invalid"}, client: &http.Client{
			Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusOK, Body: body}, nil
			}),
		}}

		version, err := rt.Version(t.Context())
		if closed != 1 || !errors.Is(err, closeErr) {
			t.Fatalf("version=%q closed=%d error=%v", version, closed, err)
		}

		if content == "invalid JSON" && !strings.Contains(err.Error(), "deserialize response data") {
			t.Fatalf("construction failure lost: %v", err)
		}
	}
}
