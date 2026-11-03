package repl

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/MontFerret/api"
	portable "github.com/MontFerret/api/diagnostics"
	"github.com/MontFerret/cli/v2/internal/testutil/encoded"
	"github.com/MontFerret/cli/v2/internal/testutil/wirehost"
	"github.com/MontFerret/cli/v2/pkg/runtime"
)

func TestWireReplConsumptionDiagnosticsAndCleanup(t *testing.T) {
	for _, cleanupFailure := range []bool{false, true} {
		t.Run(map[bool]string{false: "diagnostic", true: "cleanup"}[cleanupFailure], func(t *testing.T) {
			runs := 0
			host := wirehost.NewWithRun(t, func(context.Context, api.Source, ...api.SessionOption) (api.Output, error) {
				runs++
				output := &encoded.Output{Chunks: [][]byte{[]byte("available")}}
				if runs == 1 {
					output.Err = portable.Diagnostics{{Kind: portable.Kind("RuntimeError"), Message: "query failed"}}
					if cleanupFailure {
						output.CloseErr = errors.New("cleanup failed")
					}
				}

				return output, nil
			})
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			var stdout, stderr bytes.Buffer
			err := StartWithIO(ctx, runtime.Options{Type: "wire", Endpoint: host.Endpoint}, nil, strings.NewReader("RETURN 1\nRETURN 2\nexit\n"), &stdout, &stderr)
			if !strings.Contains(stderr.String(), "query failed") {
				t.Fatalf("diagnostic lost: %q error=%v", stderr.String(), err)
			}

			if !cleanupFailure && (err != nil || runs != 2 || strings.Count(stdout.String(), "available\n") != 2) {
				t.Fatalf("runs=%d stdout=%q error=%v", runs, stdout.String(), err)
			}

			if cleanupFailure && (err == nil || runs != 1) {
				t.Fatalf("runs=%d stderr=%q error=%v", runs, stderr.String(), err)
			}
		})
	}
}
