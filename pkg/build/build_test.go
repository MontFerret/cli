package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MontFerret/ferret/v2/pkg/bytecode/artifact"
	"github.com/MontFerret/ferret/v2/pkg/compiler"
	"github.com/MontFerret/ferret/v2/pkg/file"
)

func TestPlanOutputs_DefaultOutputPath(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "query.fql")

	plan, err := PlanOutputs([]string{input}, "")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(plan.Targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(plan.Targets))
	}

	if plan.Targets[0].OutputPath != filepath.Join(dir, "query.fqlc") {
		t.Fatalf("unexpected output path: %s", plan.Targets[0].OutputPath)
	}
}

func TestPlanOutputs_SingleFileExplicitOutput(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "query.fql")
	output := filepath.Join(dir, "compiled.bin")

	plan, err := PlanOutputs([]string{input}, output)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if plan.Targets[0].OutputPath != output {
		t.Fatalf("unexpected output path: %s", plan.Targets[0].OutputPath)
	}
}

func TestPlanOutputs_SingleFileOutputDirectory(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "query.fql")
	outputDir := filepath.Join(dir, "dist")

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		t.Fatal(err)
	}

	plan, err := PlanOutputs([]string{input}, outputDir)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if plan.OutputDir != outputDir {
		t.Fatalf("unexpected output dir: %s", plan.OutputDir)
	}

	if plan.Targets[0].OutputPath != filepath.Join(outputDir, "query.fqlc") {
		t.Fatalf("unexpected output path: %s", plan.Targets[0].OutputPath)
	}
}

func TestPlanOutputs_MultiFileOutputDirectory(t *testing.T) {
	dir := t.TempDir()
	inputA := filepath.Join(dir, "first.fql")
	inputB := filepath.Join(dir, "second")
	outputDir := filepath.Join(dir, "dist")

	plan, err := PlanOutputs([]string{inputA, inputB}, outputDir)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(plan.Targets) != 2 {
		t.Fatalf("expected 2 targets, got %d", len(plan.Targets))
	}

	if plan.Targets[0].OutputPath != filepath.Join(outputDir, "first.fqlc") {
		t.Fatalf("unexpected first output path: %s", plan.Targets[0].OutputPath)
	}

	if plan.Targets[1].OutputPath != filepath.Join(outputDir, "second.fqlc") {
		t.Fatalf("unexpected second output path: %s", plan.Targets[1].OutputPath)
	}
}

func TestPlanOutputs_MultiFileOutputMustBeDirectory(t *testing.T) {
	dir := t.TempDir()
	inputA := filepath.Join(dir, "first.fql")
	inputB := filepath.Join(dir, "second.fql")
	output := filepath.Join(dir, "artifact.fqlc")

	if err := os.WriteFile(output, []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := PlanOutputs([]string{inputA, inputB}, output)

	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "--output must be a directory when building multiple files") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPlanOutputs_MultiFileOutputCollision(t *testing.T) {
	dir := t.TempDir()
	inputA := filepath.Join(dir, "one", "query.fql")
	inputB := filepath.Join(dir, "two", "query.fql")
	outputDir := filepath.Join(dir, "dist")

	_, err := PlanOutputs([]string{inputA, inputB}, outputDir)

	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "output collision") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWriteArtifact_RejectsOverwritingSource(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "query.fql")
	query := "RETURN 42"

	writeQuery(t, input, query)

	err := WriteArtifact(compiler.New(), file.NewSource(input, query), input)

	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "would overwrite source file") {
		t.Fatalf("unexpected error: %v", err)
	}

	content, readErr := os.ReadFile(input)
	if readErr != nil {
		t.Fatal(readErr)
	}

	if string(content) != query {
		t.Fatalf("expected source file to remain unchanged, got %q", string(content))
	}
}

func TestWriteArtifact_InvalidQueryDoesNotCreateArtifact(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "broken.fql")
	output := filepath.Join(dir, "broken.fqlc")

	writeQuery(t, input, "FOR item IN")

	err := WriteArtifact(compiler.New(), file.NewSource(input, "FOR item IN"), output)

	if err == nil {
		t.Fatal("expected error")
	}

	if _, statErr := os.Stat(output); !os.IsNotExist(statErr) {
		t.Fatalf("expected artifact to be absent, stat err=%v", statErr)
	}
}

func TestWriteArtifact_CreatesMissingParentDirectory(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "query.fql")
	output := filepath.Join(dir, "nested", "out", "query.fqlc")

	writeQuery(t, input, "RETURN 42")

	if err := WriteArtifact(compiler.New(), file.NewSource(input, "RETURN 42"), output); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertArtifactSource(t, output, "RETURN 42")
}

func TestWriteArtifact_ReplacesExistingDestinationFile(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "query.fql")
	output := filepath.Join(dir, "query.fqlc")

	writeQuery(t, input, "RETURN 1")

	if err := WriteArtifact(compiler.New(), file.NewSource(input, "RETURN 1"), output); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertArtifactSource(t, output, "RETURN 1")

	writeQuery(t, input, "RETURN 2")

	if err := WriteArtifact(compiler.New(), file.NewSource(input, "RETURN 2"), output); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertArtifactSource(t, output, "RETURN 2")
}

func TestWriteArtifact_ReplacesExistingDestinationFileInNestedDirectory(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "query.fql")
	output := filepath.Join(dir, "nested", "out", "query.fqlc")

	writeQuery(t, input, "RETURN 1")

	if err := WriteArtifact(compiler.New(), file.NewSource(input, "RETURN 1"), output); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertArtifactSource(t, output, "RETURN 1")

	writeQuery(t, input, "RETURN 2")

	if err := WriteArtifact(compiler.New(), file.NewSource(input, "RETURN 2"), output); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertArtifactSource(t, output, "RETURN 2")
}

func TestWriteArtifact_ArtifactRoundTrip(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "query.fql")
	output := filepath.Join(dir, "query.fqlc")

	writeQuery(t, input, "RETURN 42")

	if err := WriteArtifact(compiler.New(), file.NewSource(input, "RETURN 42"), output); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertArtifactSource(t, output, "RETURN 42")
}

func TestWriteArtifact_InvalidQueryDoesNotCreateArtifactInMissingParentDirectory(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "broken.fql")
	output := filepath.Join(dir, "nested", "out", "broken.fqlc")

	writeQuery(t, input, "FOR item IN")

	err := WriteArtifact(compiler.New(), file.NewSource(input, "FOR item IN"), output)

	if err == nil {
		t.Fatal("expected error")
	}

	if _, statErr := os.Stat(output); !os.IsNotExist(statErr) {
		t.Fatalf("expected artifact to be absent, stat err=%v", statErr)
	}
}

func writeQuery(t *testing.T, path, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertArtifactSource(t *testing.T, path, expected string) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	program, err := artifact.Unmarshal(data)
	if err != nil {
		t.Fatalf("unmarshal artifact: %v", err)
	}

	if program.Source == nil {
		t.Fatal("expected serialized source")
	}

	if program.Source.Content() != expected {
		t.Fatalf("expected source %q, got %q", expected, program.Source.Content())
	}
}
