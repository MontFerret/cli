package run

import (
	"strings"
	"testing"

	"github.com/MontFerret/cli/v2/internal/testutil/wirehost"
)

func TestArrayDiagnosticsBuiltinAndWire(t *testing.T) {
	host := wirehost.New(t)
	for _, mode := range []string{"builtin", "wire"} {
		for _, test := range []struct {
			query string
			want  []string
		}{
			{"let arr = [1, 2, 3", []string{"SyntaxError: Unclosed array literal\n", ":1:19\n", "1 | let arr = [1, 2, 3\n  |                   ^ expected ']'\n", "Hint: Add ']' to close the array.\n"}},
			{"return [\n  \"é🙂\"", []string{"SyntaxError: Unclosed array literal\n", ":2:11\n", "2 |   \"é🙂\"\n  |       ^ expected ']'\n", "Hint: Add ']' to close the array.\n"}},
			{"return [1, 2 +", []string{"SyntaxError: Expected right-hand expression after '+'\n", "missing expression"}},
		} {
			t.Run(mode+"/"+test.query, func(t *testing.T) {
				args := []string{"run", "--runtime", mode, "--eval", test.query}
				if mode == "wire" {
					args = append(args, "--runtime-endpoint", host.Endpoint)
				}

				stdout, stderr, err := runWireCommand(t, args)
				if err == nil || stdout != "" || strings.Count(stderr, "SyntaxError:") != 1 {
					t.Fatalf("stdout=%q stderr=%q error=%v", stdout, stderr, err)
				}

				for _, want := range test.want {
					if !strings.Contains(stderr, want) {
						t.Errorf("stderr missing %q: %q", want, stderr)
					}
				}
			})
		}
	}
}
