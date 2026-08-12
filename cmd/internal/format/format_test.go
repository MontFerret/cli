package format

import (
	"path/filepath"
	"testing"

	"github.com/MontFerret/cli/v2/cmd/internal/testutil"
)

func TestFormatterCaseMode(t *testing.T) {
	tests := map[string]struct {
		input    string
		caseMode string
		want     string
	}{
		"default lower": {
			input: "RETURN 1",
			want:  "return 1",
		},
		"explicit upper": {
			input:    "return 1",
			caseMode: "upper",
			want:     "RETURN 1",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			input := filepath.Join(t.TempDir(), "query.fql")
			testutil.WriteQuery(t, input, test.input)

			cmd := New(nil)
			if err := cmd.Flags().Set("dry-run", "true"); err != nil {
				t.Fatal(err)
			}
			if test.caseMode != "" {
				if err := cmd.Flags().Set("case-mode", test.caseMode); err != nil {
					t.Fatal(err)
				}
			}

			got, err := testutil.CaptureStdout(t, func() error {
				return cmd.RunE(cmd, []string{input})
			})
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("unexpected output: got %q, want %q", got, test.want)
			}
		})
	}
}

func TestFormatterCaseModeDefault(t *testing.T) {
	cmd := New(nil)
	flag := cmd.Flags().Lookup("case-mode")
	if flag == nil {
		t.Fatal("case-mode flag is not registered")
	}
	if flag.DefValue != "lower" {
		t.Fatalf("unexpected case-mode default: got %q, want %q", flag.DefValue, "lower")
	}
}
