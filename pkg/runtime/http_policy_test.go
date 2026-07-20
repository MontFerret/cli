package runtime

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MontFerret/ferret/v2"
	ferrethttp "github.com/MontFerret/ferret/v2/pkg/net/http"
	"github.com/MontFerret/ferret/v2/pkg/source"
)

func TestBuiltinHTTPPolicyBlocksLocalhostByDefault(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("unexpected"))
	}))
	defer server.Close()

	_, err := Run(
		context.Background(),
		NewDefaultOptions(),
		source.NewAnonymous(fmt.Sprintf("RETURN IO::NET::HTTP::GET(%q)", server.URL)),
		nil,
	)
	if err == nil || !strings.Contains(err.Error(), "localhost is not allowed") {
		t.Fatalf("expected localhost policy error, got %v", err)
	}
}

func TestBuiltinHTTPPolicyAllowsConfiguredLocalhostAndSendsDefaultHeaders(t *testing.T) {
	requests := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- r.Header.Get("X-Ferret-Policy")
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	opts := NewDefaultOptions()
	opts.HTTPPolicy = []ferrethttp.PolicyOption{
		ferrethttp.WithAllowLocalhost(true),
		ferrethttp.WithDefaultHeaders(map[string]string{"X-Ferret-Policy": "configured"}),
	}

	out, err := Run(
		context.Background(),
		opts,
		source.NewAnonymous(fmt.Sprintf("RETURN TO_STRING(IO::NET::HTTP::GET(%q))", server.URL)),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()

	if got := <-requests; got != "configured" {
		t.Fatalf("expected configured default header, got %q", got)
	}
}

func TestBuiltinHTTPPolicyRejectsInvalidConfiguration(t *testing.T) {
	opts := NewDefaultOptions()
	opts.HTTPPolicy = []ferrethttp.PolicyOption{ferrethttp.WithAllowedHosts("bad host")}

	_, err := NewBuiltin(opts)
	if err == nil || !strings.Contains(err.Error(), "WithAllowedHosts") {
		t.Fatalf("expected allowed-host policy error, got %v", err)
	}
}

func TestBuiltinHTTPPolicyEnforcesResponseLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("too large"))
	}))
	defer server.Close()

	opts := NewDefaultOptions()
	opts.HTTPPolicy = []ferrethttp.PolicyOption{
		ferrethttp.WithAllowLocalhost(true),
		ferrethttp.WithMaxResponseSize(1),
	}

	_, err := Run(
		context.Background(),
		opts,
		source.NewAnonymous(fmt.Sprintf("RETURN IO::NET::HTTP::GET(%q)", server.URL)),
		nil,
	)
	if err == nil || !strings.Contains(err.Error(), "response body exceeds") {
		t.Fatalf("expected response-size error, got %v", err)
	}
}

func TestBuiltinCloseSucceedsWithConfiguredHTTPPolicy(t *testing.T) {
	opts := NewDefaultOptions()
	opts.HTTPPolicy = []ferrethttp.PolicyOption{ferrethttp.WithAllowLocalhost(true)}

	rt, err := NewBuiltin(opts)
	if err != nil {
		t.Fatal(err)
	}
	if err := rt.Close(); err != nil {
		t.Fatalf("expected close to succeed, got %v", err)
	}
}

func TestNewRejectsHTTPPolicyForRemoteRuntime(t *testing.T) {
	opts := NewDefaultOptions()
	opts.Type = "https://worker.example"
	opts.HTTPPolicy = []ferrethttp.PolicyOption{ferrethttp.WithAllowLocalhost(true)}

	_, err := New(opts)
	if !errors.Is(err, ErrHTTPPolicyRequiresBuiltinRuntime) {
		t.Fatalf("expected builtin runtime policy error, got %v", err)
	}
}

func TestDebugSessionUsesConfiguredHTTPPolicy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	opts := NewDefaultOptions()
	opts.HTTPPolicy = []ferrethttp.PolicyOption{ferrethttp.WithAllowLocalhost(true)}

	session, err := NewDebugSession(
		context.Background(),
		opts,
		nil,
		source.NewAnonymous(fmt.Sprintf("RETURN TO_STRING(IO::NET::HTTP::GET(%q))", server.URL)),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	event, err := session.Start(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if event.Reason != ferret.DebugReasonEntry {
		t.Fatalf("expected debugger entry event, got %#v", event)
	}

	event, err = session.Continue(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if event.Reason != ferret.DebugReasonCompleted {
		t.Fatalf("expected debugger completion event, got %#v", event)
	}
}
