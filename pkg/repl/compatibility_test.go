package repl

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/MontFerret/cli/v2/pkg/runtime"
)

func TestBuiltinReplBannerAndSyntaxRecovery(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	var stdout, stderr bytes.Buffer
	err := StartWithIO(ctx, runtime.NewDefaultOptions(), nil, strings.NewReader("RETURN 41\nRETURN )\nRETURN 42\nexit\n"), &stdout, &stderr)
	want := "Welcome to Ferret REPL " + runtime.EmbeddedVersion() + "\nPlease use `exit` or `Ctrl-D` to exit this program.\n41\n42\n"
	if err != nil || stdout.String() != want || !strings.Contains(stderr.String(), "anonymous:1:8") {
		t.Fatalf("stdout=%q stderr=%q error=%v", stdout.String(), stderr.String(), err)
	}
}

func TestLegacyHTTPReplBannerAndRequests(t *testing.T) {
	records := make(chan string, 3)
	worker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		records <- r.Method + " " + r.URL.Path
		if r.URL.Path == "/info" {
			_, _ = io.WriteString(w, `{"version":{"ferret":"legacy-host"}}`)

			return
		}

		_, _ = io.Copy(io.Discard, r.Body)
		_, _ = io.WriteString(w, "42")
	}))
	defer worker.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	var stdout, stderr bytes.Buffer
	err := StartWithIO(ctx, runtime.Options{Type: worker.URL}, nil, strings.NewReader("RETURN 42\nRETURN 42\nexit\n"), &stdout, &stderr)
	var requests []string
	for range 3 {
		select {
		case request := <-records:
			requests = append(requests, request)
		case <-ctx.Done():
			t.Fatal("legacy requests did not complete")
		}
	}

	want := "Welcome to Ferret REPL legacy-host\nPlease use `exit` or `Ctrl-D` to exit this program.\n42\n42\n"
	if err != nil || stdout.String() != want || stderr.Len() != 0 || strings.Join(requests, ",") != "GET /info,POST /,POST /" {
		t.Fatalf("stdout=%q stderr=%q requests=%v error=%v", stdout.String(), stderr.String(), requests, err)
	}
}
