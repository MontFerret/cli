package repl

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MontFerret/cli/v2/pkg/runtime"
)

func TestHTTPReplOpaqueAndEmptyVersions(t *testing.T) {
	for _, version := range []string{" hosted+opaque \n", ""} {
		t.Run(version, func(t *testing.T) {
			worker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/info" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL)
				}

				_ = json.NewEncoder(w).Encode(map[string]any{"version": map[string]string{"ferret": version}})
			}))
			defer worker.Close()
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			var stdout, stderr bytes.Buffer
			err := StartWithIO(ctx, runtime.Options{Type: worker.URL}, nil, strings.NewReader("exit\n"), &stdout, &stderr)
			want := "Welcome to Ferret REPL " + version + "\nPlease use `exit` or `Ctrl-D` to exit this program.\n"
			if err != nil || stdout.String() != want || stderr.Len() != 0 {
				t.Fatalf("stdout=%q stderr=%q error=%v", stdout.String(), stderr.String(), err)
			}
		})
	}
}

func TestHTTPReplTransportAndResponseFailuresEndShell(t *testing.T) {
	for _, partial := range []bool{false, true} {
		t.Run(map[bool]string{false: "request", true: "response"}[partial], func(t *testing.T) {
			var posts atomic.Int32
			worker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/info" {
					_, _ = io.WriteString(w, `{"version":{"ferret":"http-host"}}`)

					return
				}

				posts.Add(1)
				_, _ = io.Copy(io.Discard, r.Body)
				if partial {
					w.Header().Set("Content-Length", "100")
					_, _ = io.WriteString(w, "available")

					return
				}

				connection, _, err := w.(http.Hijacker).Hijack()
				if err != nil {
					t.Errorf("hijack response: %v", err)

					return
				}

				_ = connection.Close()
			}))
			defer worker.Close()
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			var stdout, stderr bytes.Buffer
			err := StartWithIO(ctx, runtime.Options{Type: worker.URL}, nil, strings.NewReader("RETURN 41\nRETURN 42\nexit\n"), &stdout, &stderr)
			want := "Welcome to Ferret REPL http-host\nPlease use `exit` or `Ctrl-D` to exit this program.\n"
			if partial {
				want += "available\n"
				if !errors.Is(err, io.ErrUnexpectedEOF) || !strings.Contains(stderr.String(), "read response data") {
					t.Fatalf("response failure lost: %v; stderr=%q", err, stderr.String())
				}
			} else if err == nil || !strings.Contains(stderr.String(), "make HTTP request to remote runtime") {
				t.Fatalf("request failure lost: %v; stderr=%q", err, stderr.String())
			}

			if stdout.String() != want || posts.Load() != 1 {
				t.Fatalf("shell continued or output changed: stdout=%q posts=%d error=%v", stdout.String(), posts.Load(), err)
			}
		})
	}
}

func TestHTTPReplExecutionCancellation(t *testing.T) {
	started, finished := make(chan struct{}), make(chan struct{})
	worker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/info" {
			_, _ = io.WriteString(w, `{"version":{"ferret":"http-host"}}`)

			return
		}

		defer close(finished)
		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		close(started)
		<-r.Context().Done()
	}))
	defer worker.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	var stdout, stderr bytes.Buffer
	done := make(chan error, 1)
	go func() {
		done <- StartWithIO(ctx, runtime.Options{Type: worker.URL}, nil, strings.NewReader("RETURN 42\n"), &stdout, &stderr)
	}()

	select {
	case <-started:
	case err := <-done:
		t.Fatalf("shell returned before query started: %v", err)
	case <-ctx.Done():
		t.Fatal("HTTP query did not start")
	}

	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) || !strings.Contains(stderr.String(), "context canceled") {
			t.Fatalf("cancellation lost: error=%v stderr=%q", err, stderr.String())
		}
	case <-time.After(10 * time.Second):
		t.Fatal("HTTP REPL did not settle")
	}

	select {
	case <-finished:
	case <-time.After(10 * time.Second):
		t.Fatal("Worker request remained active")
	}
}
