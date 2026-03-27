package cmd

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestRunBuild_MixedMultiFileBuildContinues(t *testing.T) {
	dir := t.TempDir()
	valid := filepath.Join(dir, "valid.fql")
	invalid := filepath.Join(dir, "invalid.fql")
	outputDir := filepath.Join(dir, "dist")

	writeQuery(t, valid, "RETURN 1")
	writeQuery(t, invalid, "FOR item IN")

	_, err := captureStderr(t, func() error {
		return runBuild([]string{valid, invalid}, outputDir)
	})

	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "1 of 2 scripts failed to build") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunBuild_PlanErrorReturned(t *testing.T) {
	dir := t.TempDir()
	inputA := filepath.Join(dir, "first.fql")
	inputB := filepath.Join(dir, "second.fql")
	output := filepath.Join(dir, "artifact.fqlc")

	writeQuery(t, inputA, "RETURN 1")
	writeQuery(t, inputB, "RETURN 2")
	writeQuery(t, output, "not a directory")

	_, err := captureStderr(t, func() error {
		return runBuild([]string{inputA, inputB}, output)
	})

	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "--output must be a directory when building multiple files") {
		t.Fatalf("unexpected error: %v", err)
	}
}
