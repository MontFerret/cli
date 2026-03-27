package cmd

import (
	"strings"
	"testing"
)

func TestRunBuild_MixedMultiFileBuildContinues(t *testing.T) {
	dir := t.TempDir()
	valid := dir + "/valid.fql"
	invalid := dir + "/invalid.fql"
	outputDir := dir + "/dist"

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
	inputA := dir + "/first.fql"
	inputB := dir + "/second.fql"
	output := dir + "/artifact.fqlc"

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
