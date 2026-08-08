package install

import "testing"

func TestParseInstallReference(t *testing.T) {
	tests := []struct {
		name        string
		reference   string
		wantID      string
		wantVersion string
		wantError   bool
	}{
		{name: "latest", reference: "montferret/archive", wantID: "montferret/archive"},
		{name: "explicit", reference: "montferret/archive@1.0.0-rc.3", wantID: "montferret/archive", wantVersion: "1.0.0-rc.3"},
		{name: "leading v", reference: "montferret/archive@v1.0.0", wantError: true},
		{name: "empty version", reference: "montferret/archive@", wantError: true},
		{name: "multiple separators", reference: "montferret/archive@1.0.0@next", wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, version, err := parseInstallReference(tt.reference)
			if (err != nil) != tt.wantError {
				t.Fatalf("unexpected error: %v", err)
			}
			if id != tt.wantID || version != tt.wantVersion {
				t.Fatalf("got %q@%q, want %q@%q", id, version, tt.wantID, tt.wantVersion)
			}
		})
	}
}

func TestReleaseSupportsFerret(t *testing.T) {
	version, err := parseProjectFerretVersion("v2.0.0-alpha.44")
	if err != nil {
		t.Fatal(err)
	}

	compatible, err := releaseSupportsFerret(">=2.0.0-alpha.43 <3.0.0", version)
	if err != nil || !compatible {
		t.Fatalf("expected compatible release, got compatible=%v error=%v", compatible, err)
	}
	if _, err := releaseSupportsFerret("", version); err == nil {
		t.Fatal("expected missing constraint to fail")
	}
	if _, err := releaseSupportsFerret("not a constraint", version); err == nil {
		t.Fatal("expected malformed constraint to fail")
	}
}
