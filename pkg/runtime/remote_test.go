package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"testing/iotest"
	"time"

	"github.com/MontFerret/api"
	"github.com/MontFerret/contrib/modules/web/html/drivers"
)

func TestRemoteCanonicalRun(t *testing.T) {
	params := map[string]any{"value": 1, "nothing": nil, "array": []any{true, "123"}, "object": map[string]any{"key": false}}
	wantBody := `{"text":"RETURN @value","params":{"array":[true,"123"],"nothing":null,"object":{"key":false},"quoted":"123","value":42}}`
	content := []byte{' ', '4', '2', '\n', 0, 0xff}
	var requests atomic.Int32
	worker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil || string(body) != wantBody || r.Method != http.MethodPost || r.URL.RequestURI() != "/" || r.Header.Get("Content-Type") != "application/json" || r.Header.Get("X-Worker") != "custom" {
			t.Errorf("request=%s %s headers=%v body=%q error=%v", r.Method, r.URL, r.Header, body, err)
		}

		requests.Add(1)
		w.Header().Set("Content-Type", "application/example; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write(content)
	}))
	defer worker.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	resources, err := New(ctx, Options{Type: worker.URL + "/base?ignored=yes", Headers: drivers.NewHTTPHeadersWith(map[string][]string{"X-Worker": {"custom"}})})
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = resources.Close() })
	if _, ok := resources.Runtime.(*Remote); !ok || resources.builtin != nil || resources.Disconnected != nil {
		t.Fatalf("HTTP construction did not select only Remote: %+v", resources)
	}

	handle, err := resources.Runtime.Run(ctx, api.NewSource("remote.fql", "RETURN @value"), nil,
		api.WithParams(params), api.WithParam("value", 2), api.WithParams(map[string]any{"value": 42, "quoted": "123"}), api.WithParams(nil))
	output, err := collectOutput(t, handle, err)
	if err != nil || output == nil || !bytes.Equal(output.Data, content) || output.Metadata.ContentType != "application/example; charset=utf-8" {
		t.Fatalf("output=%+v error=%v", output, err)
	}

	if params["value"] != 1 || len(params) != 4 {
		t.Fatalf("session options mutated caller parameters: %v", params)
	}

	for range 2 {
		if err := resources.Close(); err != nil {
			t.Fatal(err)
		}
	}

	// Remote.Close borrows the shared client and leaves it usable.
	handle, err = resources.Runtime.Run(ctx, api.NewSource("remote.fql", "RETURN @value"), api.WithParams(params), api.WithParams(map[string]any{"value": 42, "quoted": "123"}))
	output, err = collectOutput(t, handle, err)
	if err != nil || output == nil || !bytes.Equal(output.Data, content) {
		t.Fatalf("borrowed client became unusable: output=%v error=%v", output, err)
	}

	if got := requests.Load(); got != 2 {
		t.Fatalf("requests=%d, want 2", got)
	}
}

func TestRemoteParameterPresence(t *testing.T) {
	for _, test := range []struct {
		name, want string
		setters    []api.SessionOption
	}{
		{"omitted", "null", nil},
		{"nil", "null", []api.SessionOption{api.WithParams(nil)}},
		{"empty", "{}", []api.SessionOption{api.WithParams(map[string]any{})}},
		{"null value", `{"value":null}`, []api.SessionOption{api.WithParam("value", nil)}},
	} {
		t.Run(test.name, func(t *testing.T) {
			rt := remoteWithTransport(roundTripFunc(func(request *http.Request) (*http.Response, error) {
				body, err := io.ReadAll(request.Body)
				want := `{"text":"RETURN @value","params":` + test.want + `}`
				if err != nil || string(body) != want {
					t.Errorf("body=%q error=%v, want %q", body, err, want)
				}

				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
			}))

			handle, err := rt.Run(t.Context(), api.NewAnonymousSource("RETURN @value"), test.setters...)
			output, err := collectOutput(t, handle, err)
			if err != nil || output == nil || len(output.Data) != 0 {
				t.Fatalf("empty success: output=%v error=%v", output, err)
			}
		})
	}
}

func TestRemoteOptionsAndSerializationRejectBeforeRequest(t *testing.T) {
	firstErr, lastErr := errors.New("first option failed"), errors.New("last option failed")
	requests := 0
	rt := remoteWithTransport(roundTripFunc(func(*http.Request) (*http.Response, error) {
		requests++

		return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
	}))
	var callbacks []int
	output, err := rt.Run(t.Context(), api.NewAnonymousSource("RETURN 42"),
		func(api.SessionOptions) error {
			callbacks = append(callbacks, 1)

			return firstErr
		}, nil,
		api.WithOutputContentType("application/json"), api.WithFSRoot(""),
		func(api.SessionOptions) error {
			callbacks = append(callbacks, 2)

			return lastErr
		})
	if output != nil || !errors.Is(err, firstErr) || !errors.Is(err, lastErr) || !errors.Is(err, errors.ErrUnsupported) || !reflect.DeepEqual(callbacks, []int{1, 2}) {
		t.Fatalf("output=%v callbacks=%v error=%v", output, callbacks, err)
	}

	if !strings.Contains(err.Error(), "output codec") || !strings.Contains(err.Error(), "filesystem root") {
		t.Fatalf("unsupported setting errors were not both retained: %v", err)
	}

	for _, option := range []api.SessionOption{api.WithOutputContentType(""), api.WithFSRoot("/tmp")} {
		if output, err := rt.Run(t.Context(), api.NewAnonymousSource("RETURN 42"), option); output != nil || !errors.Is(err, errors.ErrUnsupported) {
			t.Fatalf("unsupported setting: output=%v error=%v", output, err)
		}
	}

	output, err = rt.Run(t.Context(), api.NewAnonymousSource("RETURN @value"), api.WithParam("value", make(chan int)))
	var invalid *json.UnsupportedTypeError
	if output != nil || !errors.As(err, &invalid) || !strings.Contains(err.Error(), "serialize query") || requests != 0 {
		t.Fatalf("output=%v requests=%d error=%v", output, requests, err)
	}
}

func TestRemoteCompilationUnsupportedAndCancellation(t *testing.T) {
	requests := 0
	rt := remoteWithTransport(roundTripFunc(func(*http.Request) (*http.Response, error) {
		requests++

		return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
	}))
	for _, compile := range []func(context.Context, api.Source, ...api.PlanOption) (api.Plan, error){rt.Compile, rt.CompileDebug} {
		plan, err := compile(t.Context(), api.NewAnonymousSource("RETURN )"))
		if plan != nil || !errors.Is(err, errors.ErrUnsupported) || !strings.Contains(err.Error(), "HTTP Worker") || !strings.Contains(err.Error(), "builtin") {
			t.Fatalf("plan=%v error=%v", plan, err)
		}
	}

	for _, expected := range []error{context.Canceled, context.DeadlineExceeded} {
		ctx, cancel := context.WithCancel(t.Context())
		if expected == context.DeadlineExceeded {
			cancel()
			ctx, cancel = context.WithDeadline(t.Context(), time.Time{})
		} else {
			cancel()
		}

		for _, compile := range []func(context.Context, api.Source, ...api.PlanOption) (api.Plan, error){rt.Compile, rt.CompileDebug} {
			if plan, err := compile(ctx, api.NewAnonymousSource("RETURN 42")); plan != nil || !errors.Is(err, expected) {
				t.Fatalf("canceled compilation: plan=%v error=%v", plan, err)
			}
		}

		if version, err := rt.Version(ctx); version != "" || !errors.Is(err, expected) {
			t.Fatalf("canceled metadata: version=%q error=%v", version, err)
		}

		called := false
		output, err := rt.Run(ctx, api.NewAnonymousSource("RETURN 42"), func(api.SessionOptions) error {
			called = true

			return nil
		})
		cancel()
		if output != nil || !errors.Is(err, expected) || called {
			t.Fatalf("canceled run: output=%v callback=%v error=%v", output, called, err)
		}
	}

	if requests != 0 {
		t.Fatalf("unsupported or canceled work made %d requests", requests)
	}
}

func TestRemoteResponseOutputAndFailures(t *testing.T) {
	readErr, closeErr := errors.New("response read failed"), errors.New("response close failed")
	for _, test := range []struct {
		name, content string
		read, close   error
		output        bool
	}{
		{"bytes", " \n42\n", nil, nil, true},
		{"empty", "", nil, nil, true},
		{"partial", "available", readErr, closeErr, true},
		{"no output", "", readErr, closeErr, false},
		{"close only", "42", nil, closeErr, true},
		{"empty close", "", nil, closeErr, true},
	} {
		t.Run(test.name, func(t *testing.T) {
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

				return test.close
			})}
			rt := remoteWithTransport(roundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/opaque"}}, Body: body, ContentLength: -1}, nil
			}))
			handle, err := rt.Run(t.Context(), api.NewAnonymousSource("RETURN 42"))
			output, err := collectOutput(t, handle, err)
			if (output != nil) != test.output {
				t.Fatalf("output presence changed: output=%v error=%v", output, err)
			}

			for _, failure := range []error{test.read, test.close} {
				if failure != nil && !errors.Is(err, failure) {
					t.Fatalf("failure %v lost: %v", failure, err)
				}
			}

			if test.read == nil && test.close == nil && err != nil {
				t.Fatalf("output=%v error=%v", output, err)
			}

			if output != nil && (string(output.Data) != test.content || output.Metadata.ContentType != "application/opaque") {
				t.Fatalf("output changed: %+v", output)
			}

			for range 2 {
				if err := rt.Close(); err != nil {
					t.Fatal(err)
				}
			}

			if closed != 1 {
				t.Fatalf("response closed %d times, want 1", closed)
			}
		})
	}
}

func TestRemoteResponseCancellation(t *testing.T) {
	for _, metadata := range []bool{false, true} {
		t.Run(map[bool]string{false: "run", true: "version"}[metadata], func(t *testing.T) {
			started, finished := make(chan struct{}), make(chan struct{})
			worker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				defer close(finished)
				_, _ = io.Copy(io.Discard, r.Body)
				_, _ = io.WriteString(w, "prefix")
				w.(http.Flusher).Flush()
				close(started)
				<-r.Context().Done()
			}))
			defer worker.Close()
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			resources, err := New(ctx, Options{Type: worker.URL})
			if err != nil {
				t.Fatal(err)
			}

			t.Cleanup(func() { _ = resources.Close() })
			done := make(chan error, 1)
			go func() {
				if metadata {
					_, err := resources.Runtime.Version(ctx)
					done <- err

					return
				}

				handle, err := resources.Runtime.Run(ctx, api.NewAnonymousSource("RETURN 42"))
				_, err = collectOutput(t, handle, err)
				done <- err
			}()

			select {
			case <-started:
			case err := <-done:
				t.Fatalf("request returned before response started: %v", err)
			case <-ctx.Done():
				t.Fatal("response did not start")
			}

			cancel()
			select {
			case err := <-done:
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("cancellation identity lost: %v", err)
				}
			case <-time.After(10 * time.Second):
				t.Fatal("response read did not stop")
			}

			waitSignal(t, finished)
		})
	}
}

func remoteWithTransport(transport http.RoundTripper) *Remote {
	return &Remote{url: url.URL{Scheme: "http", Host: "worker.invalid"}, client: &http.Client{Transport: transport}}
}
