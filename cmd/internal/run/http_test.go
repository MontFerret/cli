package run

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/MontFerret/cli/v2/cmd/internal/testutil"
)

func TestHTTPRunCommands(t *testing.T) {
	const query = "RETURN [@value, @nothing, @quoted]"
	const body = `{"text":"RETURN [@value, @nothing, @quoted]","params":{"nothing":null,"quoted":"123","value":{"nested":[true,42]}}}`
	const result = ` [{"nested":[true,42]},null,"123"] `
	var requests atomic.Int32
	worker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(r.Body)
		if err != nil || string(data) != body || r.Method != http.MethodPost || r.URL.Path != "/" {
			t.Errorf("request=%s %s body=%q error=%v", r.Method, r.URL, data, err)
		}

		requests.Add(1)
		_, _ = io.WriteString(w, result)
	}))
	defer worker.Close()

	for _, mode := range []string{"inline", "file", "exec", "stdin"} {
		t.Run(mode, func(t *testing.T) {
			args := []string{"run", "--runtime", worker.URL, "--param", `value={"nested":[true,42]}`, "--param", "nothing=null", "--param", `quoted="123"`}
			switch mode {
			case "inline":
				args = append(args, "--eval", query)
			case "exec":
				args[0] = "exec"
				args = append(args, "--eval", query)
			case "file":
				path := filepath.Join(t.TempDir(), "query.fql")
				testutil.WriteQuery(t, path, query)
				args = append(args, path)
			}

			invoke := func() {
				stdout, stderr, err := runWireCommand(t, args)
				if err != nil || stdout != result || stderr != "" {
					t.Fatalf("stdout=%q stderr=%q error=%v", stdout, stderr, err)
				}
			}

			if mode == "stdin" {
				testutil.WithStdinBytes(t, []byte(query), invoke)
			} else {
				invoke()
			}
		})
	}

	if got := requests.Load(); got != 4 {
		t.Fatalf("requests=%d, want 4", got)
	}
}

func TestHTTPRunEmptyAndPartialOutput(t *testing.T) {
	for _, partial := range []bool{false, true} {
		t.Run(map[bool]string{false: "empty", true: "partial"}[partial], func(t *testing.T) {
			worker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.Copy(io.Discard, r.Body)
				if partial {
					w.Header().Set("Content-Length", "100")
					_, _ = io.WriteString(w, "available")
				}
			}))
			defer worker.Close()
			stdout, stderr, err := runWireCommand(t, []string{"run", "--runtime", worker.URL, "--eval", "RETURN 42"})
			if partial {
				if stdout != "available" || !errors.Is(err, io.ErrUnexpectedEOF) || !strings.Contains(stderr, "read response data") || !strings.Contains(stderr, "unexpected EOF") {
					t.Fatalf("partial output lost: stdout=%q stderr=%q error=%v", stdout, stderr, err)
				}
			} else if stdout != "" || stderr != "" || err != nil {
				t.Fatalf("empty output changed: stdout=%q stderr=%q error=%v", stdout, stderr, err)
			}
		})
	}
}
