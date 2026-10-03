package runtime

import (
	"context"
	"errors"
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MontFerret/api"
	"github.com/MontFerret/cli/v2/internal/testutil/wirehost"
)

func TestSourceRuntimeVersion(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	builtin, err := New(ctx, NewDefaultOptions())
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if err := builtin.Close(); err != nil {
			t.Error(err)
		}
	})
	got, err := builtin.Runtime.Version(ctx)
	if err != nil || got.String() != EmbeddedVersion() {
		t.Fatalf("builtin version=%q, %v; want %q", got, err, EmbeddedVersion())
	}

	for _, value := range []api.Version{"v2.0.0-alpha.57", " runtime+opaque \n", "版本-α", "runtime\x00\xff", ""} {
		t.Run(string(value), func(t *testing.T) {
			host := wirehost.NewWithVersion(t, value)
			resources, err := New(ctx, Options{Type: "wire", Endpoint: host.Endpoint})
			if err != nil {
				t.Fatal(err)
			}

			t.Cleanup(func() {
				if err := resources.Close(); err != nil {
					t.Error(err)
				}
			})
			for range 2 {
				got, err := resources.Runtime.Version(ctx)
				if err != nil || got != value {
					t.Fatalf("hosted version=%q, %v; want %q", got, err, value)
				}
			}

			if err := resources.Close(); err != nil {
				t.Fatal(err)
			}

			got, err := resources.Runtime.Version(ctx)
			if err != nil || got != value || host.Connections() != 1 {
				t.Fatalf("metadata after cleanup=%q, %v; connections=%d", got, err, host.Connections())
			}
		})
	}
}

func TestSourcePlanParameterMetadata(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	for _, mode := range []string{"builtin", "wire"} {
		t.Run(mode, func(t *testing.T) {
			opts := NewDefaultOptions()
			if mode == "wire" {
				host := wirehost.New(t)
				opts.Type, opts.Endpoint = mode, host.Endpoint
			}

			resources, err := New(ctx, opts)
			if err != nil {
				t.Fatal(err)
			}

			t.Cleanup(func() {
				if err := resources.Close(); err != nil {
					t.Error(err)
				}
			})
			canceled, cancel := context.WithCancel(ctx)
			cancel()
			expired, expire := context.WithDeadline(ctx, time.Now().Add(-time.Second))
			defer expire()

			type parameterSnapshot struct {
				plan   api.Plan
				params []string
			}

			var plans []parameterSnapshot

			for _, debug := range []bool{false, true} {
				for _, query := range []string{"RETURN 1", "RETURN [@second, @first, @second]"} {
					compile := resources.Runtime.Compile
					if debug {
						compile = resources.Runtime.CompileDebug
					}

					plan, err := compile(ctx, api.NewSource("params.fql", query))
					if err != nil {
						t.Fatal(err)
					}

					t.Cleanup(func() {
						if err := plan.Close(); err != nil {
							t.Error(err)
						}
					})

					params, err := plan.Params(ctx)
					if err != nil {
						t.Fatal(err)
					}

					want := []string(nil)
					if query != "RETURN 1" {
						want = []string{"first", "second"}
					}

					snapshot := slices.Clone(params)
					plans = append(plans, parameterSnapshot{plan, snapshot})
					slices.Sort(params)
					if !slices.Equal(params, want) {
						t.Fatalf("Params=%v, want %v", params, want)
					}

					if len(params) != 0 {
						params[0] = "caller mutation"
					}

					again, err := plan.Params(ctx)
					if err != nil || !slices.Equal(again, snapshot) {
						t.Fatalf("parameter snapshot changed=%v, %v; want %v", again, err, snapshot)
					}

					for _, check := range []struct {
						ctx context.Context
						err error
					}{{nil, nil}, {canceled, context.Canceled}, {expired, context.DeadlineExceeded}} {
						if _, err := plan.Params(check.ctx); err == nil || (check.err != nil && !errors.Is(err, check.err)) {
							t.Fatalf("Params context error=%v; want %v", err, check.err)
						}

						if _, err := resources.Runtime.Version(check.ctx); err == nil || (check.err != nil && !errors.Is(err, check.err)) {
							t.Fatalf("Version context error=%v; want %v", err, check.err)
						}
					}

					if err := plan.Close(); err != nil {
						t.Fatal(err)
					}
				}
			}

			if err := resources.Close(); err != nil {
				t.Fatal(err)
			}

			for _, snapshot := range plans {
				params, err := snapshot.plan.Params(ctx)
				if err != nil || !slices.Equal(params, snapshot.params) {
					t.Fatalf("metadata after cleanup=%v, %v; want %v", params, err, snapshot.params)
				}
			}
		})
	}
}

func TestWireVersionHandshakeCancellation(t *testing.T) {
	for _, timedOut := range []bool{false, true} {
		t.Run(map[bool]string{false: "parent cancellation", true: "connect timeout"}[timedOut], func(t *testing.T) {
			started, finished := make(chan struct{}), make(chan struct{})
			host := wirehost.NewWithVersionFunc(t, func(ctx context.Context) (api.Version, error) {
				close(started)
				defer close(finished)
				<-ctx.Done()

				return "", ctx.Err()
			})
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			opts := Options{Type: "wire", Endpoint: host.Endpoint, ConnectTimeout: 10 * time.Second}
			if timedOut {
				opts.ConnectTimeout = 250 * time.Millisecond
			}

			done := make(chan error, 1)
			go func() {
				resources, err := New(ctx, opts)
				if resources != nil {
					err = errors.Join(err, errors.New("canceled handshake returned resources"), resources.Close())
				}

				done <- err
			}()
			waitSignal(t, started)

			if !timedOut {
				cancel()
			}

			select {
			case err := <-done:
				want := context.Canceled
				if timedOut {
					want = context.DeadlineExceeded
				}

				if !errors.Is(err, want) {
					t.Fatalf("metadata construction error=%v; want %v", err, want)
				}
			case <-time.After(10 * time.Second):
				t.Fatal("hosted Version did not unblock construction")
			}

			waitSignal(t, finished)
			cleanupCtx, cleanupCancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cleanupCancel()

			if err := host.WaitForClientsClosed(cleanupCtx); err != nil {
				t.Fatalf("partial construction leaked the transport: %v", err)
			}
		})
	}
}

func TestWireVersionSnapshotAfterTransportLoss(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	var calls atomic.Int32
	host := wirehost.NewWithVersionFunc(t, func(context.Context) (api.Version, error) {
		calls.Add(1)

		return "hosted-opaque-version", nil
	})
	resources, err := New(ctx, Options{Type: "wire", Endpoint: host.Endpoint})
	if err != nil {
		t.Fatal(err)
	}

	host.StopTransport()
	waitSignal(t, resources.Disconnected)

	for range 2 {
		version, err := resources.Runtime.Version(ctx)
		if err != nil || version != "hosted-opaque-version" {
			t.Fatalf("cached metadata after transport loss=%q, %v", version, err)
		}
	}

	first, second := resources.Close(), resources.Close()
	if (first == nil) != (second == nil) || (first != nil && first.Error() != second.Error()) {
		t.Fatalf("cleanup result was not retained: %v / %v", first, second)
	}

	if calls.Load() != 1 || host.Connections() != 1 {
		t.Fatalf("metadata reconnected or queried the host: calls=%d connections=%d", calls.Load(), host.Connections())
	}
}
