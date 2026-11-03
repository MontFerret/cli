package version

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MontFerret/cli/v2/cmd/internal/testutil"
)

func TestHTTPVersionOpaqueAndEmptyMetadata(t *testing.T) {
	for _, version := range []string{" runtime+opaque \n", "版本-α", "runtime\x00", ""} {
		t.Run(version, func(t *testing.T) {
			worker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/info" {
					t.Errorf("unexpected metadata request: %s %s", r.Method, r.URL)
				}

				_ = json.NewEncoder(w).Encode(map[string]any{"version": map[string]string{"ferret": version}})
			}))
			defer worker.Close()
			stdout, stderr, err := executeVersion(t.Context(), testutil.NewStore(t), []string{"--runtime", worker.URL})
			want := fmt.Sprintf("Version:\n  Self: test\n  Runtime: %s\n", version)
			if err != nil || stdout != want || stderr != "" {
				t.Fatalf("stdout=%q stderr=%q error=%v, want %q", stdout, stderr, err, want)
			}
		})
	}
}
