package cmd

import (
	"testing"
)

func TestParseParams_ValidJSON(t *testing.T) {
	flags := []string{
		`name:"John"`,
		`age:30`,
		`active:true`,
		`tags:["a","b"]`,
	}

	params, err := parseParams(flags)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if params["name"] != "John" {
		t.Errorf("expected name=John, got %v", params["name"])
	}

	if params["age"] != float64(30) {
		t.Errorf("expected age=30, got %v", params["age"])
	}

	if params["active"] != true {
		t.Errorf("expected active=true, got %v", params["active"])
	}
}

func TestParseParams_MissingSeparator(t *testing.T) {
	flags := []string{"invalid"}

	_, err := parseParams(flags)

	if err == nil {
		t.Fatal("expected error for missing separator")
	}
}

func TestParseParams_InvalidJSON(t *testing.T) {
	flags := []string{"key:not-valid-json"}

	_, err := parseParams(flags)

	if err == nil {
		t.Fatal("expected error for invalid JSON value")
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

func TestParseParams_ColonInValue(t *testing.T) {
	flags := []string{`url:"http://example.com"`}

	params, err := parseParams(flags)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if params["url"] != "http://example.com" {
		t.Errorf("expected url=http://example.com, got %v", params["url"])
	}
}
