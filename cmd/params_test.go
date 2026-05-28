package cmd

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseParams(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  any
	}{
		{name: "raw string", input: "name=Steve", want: "Steve"},
		{name: "raw string URL", input: "url=https://example.com", want: "https://example.com"},
		{name: "raw string with colon", input: "time=10:30", want: "10:30"},
		{name: "JSON number", input: "age=42", want: float64(42)},
		{name: "JSON bool", input: "active=true", want: true},
		{name: "JSON null", input: "missing=null", want: nil},
		{name: "explicit JSON string", input: `name="Steve"`, want: "Steve"},
		{name: "explicit numeric string", input: `code="123"`, want: "123"},
		{name: "explicit boolean string", input: `enabled="false"`, want: "false"},
		{name: "JSON array", input: `tags=["admin","editor"]`, want: []any{"admin", "editor"}},
		{name: "JSON object", input: `user={"name":"Ada"}`, want: map[string]any{"name": "Ada"}},
		{name: "backward compatible JSON string", input: `name:"Steve"`, want: "Steve"},
		{name: "backward compatible raw string", input: "name:Steve", want: "Steve"},
		{name: "invalid JSON falls back to string", input: "key:not-valid-json", want: "not-valid-json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params, err := parseParams([]string{tt.input})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			key, _, _ := strings.Cut(tt.input, "=")
			if key == tt.input {
				key, _, _ = strings.Cut(tt.input, ":")
			}
			key = strings.TrimSpace(key)

			if got := params[key]; !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("expected %v (%T), got %v (%T)", tt.want, tt.want, got, got)
			}
		})
	}
}

func TestParseParams_Empty(t *testing.T) {
	params, err := parseParams(nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(params) != 0 {
		t.Errorf("expected empty params, got %d", len(params))
	}
}

func TestParseParams_InvalidInput(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError string
	}{
		{name: "missing separator", input: "name", wantError: `invalid param "name": expected name=value`},
		{name: "empty equals name", input: "=Steve", wantError: `invalid param "=Steve": parameter name cannot be empty`},
		{name: "empty colon name", input: ":Steve", wantError: `invalid param ":Steve": parameter name cannot be empty`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseParams([]string{tt.input})
			if err == nil {
				t.Fatal("expected error")
			}

			if err.Error() != tt.wantError {
				t.Fatalf("expected error %q, got %q", tt.wantError, err.Error())
			}
		})
	}
}
