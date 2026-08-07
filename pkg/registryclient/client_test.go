package registryclient

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientNavigatesRegistryDistribution(t *testing.T) {
	server := newRegistryServer(t, nil)
	defer server.Close()

	client, err := New(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}

	catalog, err := client.Catalog(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Modules) != 1 || catalog.Modules[0].ID != "montferret/sqlite" {
		t.Fatalf("unexpected catalog: %#v", catalog)
	}

	module, err := client.Module(context.Background(), catalog.Modules[0].Href)
	if err != nil {
		t.Fatal(err)
	}
	if module.Latest != "" || len(module.Versions) != 1 {
		t.Fatalf("unexpected module: %#v", module)
	}

	version, err := client.Version(context.Background(), module.Versions[0].Href)
	if err != nil {
		t.Fatal(err)
	}
	if version.Namespace != "DB::SQLITE" || version.Documentation != server.URL+"/modules/montferret/sqlite/versions/1.0.0-rc.1/docs.md" {
		t.Fatalf("unexpected version: %#v", version)
	}
}

func TestClientClassifiesFailures(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		body    string
		status  int
		call    func(*Client) error
		wantErr error
	}{
		{
			name:    "unavailable",
			path:    "/",
			status:  http.StatusServiceUnavailable,
			call:    func(client *Client) error { _, err := client.Root(context.Background()); return err },
			wantErr: ErrUnavailable,
		},
		{
			name:    "malformed json",
			path:    "/",
			body:    `{`,
			status:  http.StatusOK,
			call:    func(client *Client) error { _, err := client.Root(context.Background()); return err },
			wantErr: ErrMalformed,
		},
		{
			name:    "trailing document",
			path:    "/",
			body:    `{"schemaVersion":1,"artifacts":{"modules":"/modules/index.json"}} {}`,
			status:  http.StatusOK,
			call:    func(client *Client) error { _, err := client.Root(context.Background()); return err },
			wantErr: ErrMalformed,
		},
		{
			name:    "unsupported schema",
			path:    "/",
			body:    `{"schemaVersion":2,"artifacts":{"modules":"/modules/index.json"}}`,
			status:  http.StatusOK,
			call:    func(client *Client) error { _, err := client.Root(context.Background()); return err },
			wantErr: ErrMalformed,
		},
		{
			name:    "not found",
			path:    "/missing.json",
			status:  http.StatusNotFound,
			call:    func(client *Client) error { _, err := client.Module(context.Background(), "/missing.json"); return err },
			wantErr: ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				if request.URL.Path != tt.path {
					http.NotFound(writer, request)
					return
				}
				writer.WriteHeader(tt.status)
				fmt.Fprint(writer, tt.body)
			}))
			defer server.Close()

			client, err := New(server.URL, server.Client())
			if err != nil {
				t.Fatal(err)
			}

			if err := tt.call(client); !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestClientRejectsUnsafeAndOversizedResponses(t *testing.T) {
	t.Run("foreign artifact", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			fmt.Fprint(writer, `{"schemaVersion":1,"artifacts":{"modules":"https://elsewhere.example/modules.json"}}`)
		}))
		defer server.Close()

		client, _ := New(server.URL, server.Client())
		_, err := client.Root(context.Background())
		if !errors.Is(err, ErrMalformed) {
			t.Fatalf("expected malformed link, got %v", err)
		}
	})

	t.Run("oversized body", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			fmt.Fprint(writer, strings.Repeat("x", 33))
		}))
		defer server.Close()

		client, _ := New(server.URL, server.Client())
		client.maxBody = 32
		_, err := client.Root(context.Background())
		if !errors.Is(err, ErrMalformed) {
			t.Fatalf("expected oversized response error, got %v", err)
		}
	})
}

func TestClientClassifiesNetworkFailure(t *testing.T) {
	server := newRegistryServer(t, nil)
	client, err := New(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	server.Close()

	_, err = client.Root(context.Background())
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("expected unavailable network error, got %v", err)
	}
}

func TestNewRejectsInvalidBaseURL(t *testing.T) {
	for _, value := range []string{"registry.example", "ftp://registry.example", "https://user@registry.example"} {
		if _, err := New(value, nil); err == nil {
			t.Fatalf("expected %q to be rejected", value)
		}
	}
}

func newRegistryServer(t *testing.T, overrides map[string]string) *httptest.Server {
	t.Helper()

	documents := map[string]string{
		"/":                                     `{"schemaVersion":1,"artifacts":{"modules":"/modules/index.json"}}`,
		"/modules/index.json":                   `{"schemaVersion":1,"modules":[{"id":"montferret/sqlite","href":"/modules/montferret/sqlite/index.json"}]}`,
		"/modules/montferret/sqlite/index.json": `{"schemaVersion":1,"id":"montferret/sqlite","owner":"montferret","name":"sqlite","description":"SQLite module","versions":[{"version":"1.0.0-rc.1","href":"/modules/montferret/sqlite/versions/1.0.0-rc.1/index.json"}]}`,
		"/modules/montferret/sqlite/versions/1.0.0-rc.1/index.json": `{"schemaVersion":1,"id":"montferret/sqlite","version":"1.0.0-rc.1","namespace":"DB::SQLITE","source":{"repository":"https://github.com/MontFerret/contrib","path":"modules/db/sqlite","commit":"0123456789abcdef0123456789abcdef01234567"},"content":{"documentation":"./docs.md"}}`,
	}
	for path, document := range overrides {
		documents[path] = document
	}

	return httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		document, exists := documents[request.URL.Path]
		if !exists {
			http.NotFound(writer, request)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, document)
	}))
}
