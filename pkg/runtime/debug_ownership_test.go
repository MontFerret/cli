package runtime

import (
	"context"
	"errors"
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/MontFerret/api"
	"github.com/MontFerret/ferret/v2"
	"github.com/MontFerret/ferret/v2/uapi"
)

func TestDebugSessionOwnership(t *testing.T) {
	compileErr := errors.New("compiler failed")
	sessionErr := errors.New("session close failed")
	planErr := errors.New("plan close failed")
	runtimeErr := errors.New("runtime close failed")
	for _, stage := range []string{"compile", "create session", "close"} {
		t.Run(stage, func(t *testing.T) {
			var closed []string
			engine, err := ferret.New(
				ferret.WithBeforeCompileHook(func(context.Context) error {
					if stage == "compile" {
						return compileErr
					}

					return nil
				}),
				ferret.WithSessionCloseHook(func() error {
					closed = append(closed, "session")
					return sessionErr
				}),
				ferret.WithPlanCloseHook(func() error {
					closed = append(closed, "plan")
					return planErr
				}),
			)
			if err != nil {
				t.Fatal(err)
			}

			defer engine.Close()
			rt := uapi.Wrap(engine, "test")
			resources := &Resources{Runtime: rt, owners: []io.Closer{closeFunc(func() error {
				closed = append(closed, "runtime")
				return errors.Join(rt.Close(), runtimeErr)
			})}}
			var params map[string]any
			if stage == "create session" {
				params = map[string]any{"invalid": make(chan int)}
			}

			session, err := newDebugSession(t.Context(), resources, "", params, api.NewSource("query.fql", "RETURN 1"))
			want := []string{"runtime"}
			switch stage {
			case "compile":
				if session != nil || !errors.Is(err, compileErr) {
					t.Fatalf("compile failure: session=%v err=%v", session, err)
				}
			case "create session":
				want = []string{"plan", "runtime"}
				if session != nil || err == nil || !strings.Contains(err.Error(), "convert session params") || !errors.Is(err, planErr) {
					t.Fatalf("session creation failure: session=%v err=%v", session, err)
				}
			case "close":
				want = []string{"session", "plan", "runtime"}
				if err != nil {
					t.Fatal(err)
				}

				for range 2 {
					err = session.Close()
					if !errors.Is(err, sessionErr) || !errors.Is(err, planErr) || !errors.Is(err, runtimeErr) {
						t.Fatalf("lost cleanup failures: %v", err)
					}
				}
			}

			if !errors.Is(err, runtimeErr) || !slices.Equal(closed, want) {
				t.Fatalf("stage=%s closed=%v want=%v err=%v", stage, closed, want, err)
			}
		})
	}
}
