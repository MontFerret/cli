package repl

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/MontFerret/api"
	"github.com/MontFerret/cli/v2/internal/testutil/wirehost"
	"github.com/MontFerret/cli/v2/pkg/runtime"
)

func TestWireReplHostedVersion(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	for _, version := range []api.Version{" runtime+opaque \n", "版本-α", ""} {
		t.Run(string(version), func(t *testing.T) {
			host := wirehost.NewWithVersion(t, version)
			var stdout, stderr bytes.Buffer
			err := StartWithIO(ctx, runtime.Options{Type: "wire", Endpoint: host.Endpoint}, nil,
				strings.NewReader("RETURN DEMO::IDENTITY(41)\nRETURN DEMO::IDENTITY(42)\nexit\n"), &stdout, &stderr)
			want := fmt.Sprintf("Welcome to Ferret REPL (Wire runtime: %s; version: %s)\nPlease use `exit` or `Ctrl-D` to exit this program.\n41\n42\n", host.Endpoint, version)
			if err != nil || stdout.String() != want || stderr.Len() != 0 || host.Connections() != 1 {
				t.Fatalf("stdout=%q stderr=%q error=%v connections=%d; want %q", stdout.String(), stderr.String(), err, host.Connections(), want)
			}
		})
	}
}

func TestWireReplMetadataFailureDoesNotReadInput(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	host := wirehost.NewWithVersionFunc(t, func(context.Context) (api.Version, error) {
		return "", errors.New("hosted metadata failed")
	})
	var stdout, stderr bytes.Buffer
	input := readFunc(func([]byte) (int, error) {
		t.Error("requested input after failed metadata")

		return 0, errors.New("unexpected input")
	})
	err := StartWithIO(ctx, runtime.Options{Type: "wire", Endpoint: host.Endpoint}, nil, input, &stdout, &stderr)
	if err == nil || stdout.Len() != 0 || stderr.Len() != 0 || host.Connections() != 1 {
		t.Fatalf("metadata failure: stdout=%q stderr=%q error=%v connections=%d", stdout.String(), stderr.String(), err, host.Connections())
	}
}

func TestWireReplBannerWriteFailureReleasesClient(t *testing.T) {
	for _, failAt := range []int{1, 2} {
		t.Run(fmt.Sprint(failAt), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			host := wirehost.New(t)
			writeErr := errors.New("banner output is closed")
			writes := 0
			output := writeFunc(func(data []byte) (int, error) {
				writes++
				if writes == failAt {
					return 0, writeErr
				}

				return len(data), nil
			})
			input := readFunc(func([]byte) (int, error) {
				t.Error("requested input after banner output failure")

				return 0, errors.New("unexpected input")
			})
			var stderr bytes.Buffer
			err := StartWithIO(ctx, runtime.Options{Type: "wire", Endpoint: host.Endpoint}, nil, input, output, &stderr)
			if !errors.Is(err, writeErr) || writes != failAt || stderr.Len() != 0 {
				t.Fatalf("banner failure: writes=%d stderr=%q error=%v", writes, stderr.String(), err)
			}

			if err := host.WaitForClientsClosed(ctx); err != nil {
				t.Fatalf("banner failure leaked client transport: %v", err)
			}

			if _, err := host.Runtime.Run(ctx, api.NewSource("survivor.fql", "RETURN DEMO::IDENTITY(42)")); err != nil {
				t.Fatalf("banner failure closed the host runtime: %v", err)
			}
		})
	}
}
