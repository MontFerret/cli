package runtime

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"testing/iotest"
)

func TestLegacyVersionClosesResponseAndRetainsFailure(t *testing.T) {
	closeErr, readErr := errors.New("close version response failed"), errors.New("read version response failed")
	for _, test := range []struct {
		content string
		read    error
	}{
		{`{"version":{"ferret":"legacy"}}`, nil}, {"invalid JSON", nil}, {`{"version":`, readErr},
	} {
		closed := 0
		var reader io.Reader = strings.NewReader(test.content)
		if test.read != nil {
			reader = io.MultiReader(reader, iotest.ErrReader(test.read))
		}

		body := &struct {
			io.Reader
			io.Closer
		}{reader, closeFunc(func() error {
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

		if test.content == `{"version":{"ferret":"legacy"}}` && version != "legacy" {
			t.Fatalf("metadata lost alongside cleanup error: %q", version)
		}

		if test.content == "invalid JSON" && !strings.Contains(err.Error(), "deserialize response data") {
			t.Fatalf("construction failure lost: %v", err)
		}

		if test.read != nil && (!errors.Is(err, test.read) || version != "") {
			t.Fatalf("metadata read failure lost: version=%q error=%v", version, err)
		}
	}
}
