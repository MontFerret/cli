package build

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MontFerret/ferret/v2/pkg/bytecode/artifact"
	"github.com/MontFerret/ferret/v2/pkg/compiler"
	"github.com/MontFerret/ferret/v2/pkg/source"
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

func TestPlanOutputs_DefaultOutputPathsMultiFile(t *testing.T) {
	dir := t.TempDir()
	inputA := filepath.Join(dir, "first.fql")
	inputB := filepath.Join(dir, "second.txt")

	plan, err := PlanOutputs([]string{inputA, inputB}, "")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(plan.Targets) != 2 {
		t.Fatalf("expected 2 targets, got %d", len(plan.Targets))
	}

	if plan.Targets[0].OutputPath != filepath.Join(dir, "first.fqlc") {
		t.Fatalf("unexpected first output path: %s", plan.Targets[0].OutputPath)
	}

	if plan.Targets[1].OutputPath != filepath.Join(dir, "second.fqlc") {
		t.Fatalf("unexpected second output path: %s", plan.Targets[1].OutputPath)
	}
}

func TestPlanOutputs_DefaultOutputCollisionDifferentExtensions(t *testing.T) {
	dir := t.TempDir()
	inputA := filepath.Join(dir, "query.fql")
	inputB := filepath.Join(dir, "query.txt")

	_, err := PlanOutputs([]string{inputA, inputB}, "")

	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "output collision") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPlanOutputs_DefaultOutputCollisionExtensionlessAndExtensioned(t *testing.T) {
	dir := t.TempDir()
	inputA := filepath.Join(dir, "query")
	inputB := filepath.Join(dir, "query.fql")

	_, err := PlanOutputs([]string{inputA, inputB}, "")

	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "output collision") {
		t.Fatalf("unexpected error: %v", err)
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

	err := WriteArtifact(t.Context(), newCompiler(t), source.New(input, query), input)

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

	err := WriteArtifact(t.Context(), newCompiler(t), source.New(input, "FOR item IN"), output)

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

	if err := WriteArtifact(t.Context(), newCompiler(t), source.New(input, "RETURN 42"), output); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertArtifactSource(t, output, "RETURN 42")
}

func TestWriteArtifact_ReplacesExistingDestinationFile(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "query.fql")
	output := filepath.Join(dir, "query.fqlc")

	writeQuery(t, input, "RETURN 1")

	if err := WriteArtifact(t.Context(), newCompiler(t), source.New(input, "RETURN 1"), output); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertArtifactSource(t, output, "RETURN 1")

	writeQuery(t, input, "RETURN 2")

	if err := WriteArtifact(t.Context(), newCompiler(t), source.New(input, "RETURN 2"), output); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertArtifactSource(t, output, "RETURN 2")
}

func TestWriteArtifact_ReplacesExistingDestinationFileInNestedDirectory(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "query.fql")
	output := filepath.Join(dir, "nested", "out", "query.fqlc")

	writeQuery(t, input, "RETURN 1")

	if err := WriteArtifact(t.Context(), newCompiler(t), source.New(input, "RETURN 1"), output); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertArtifactSource(t, output, "RETURN 1")

	writeQuery(t, input, "RETURN 2")

	if err := WriteArtifact(t.Context(), newCompiler(t), source.New(input, "RETURN 2"), output); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertArtifactSource(t, output, "RETURN 2")
}

func TestWriteArtifact_RenameFailurePreservesExistingDestinationAndCleansTemp(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "query.fql")
	output := filepath.Join(dir, "query.fqlc")

	writeQuery(t, input, "RETURN 1")

	if err := WriteArtifact(t.Context(), newCompiler(t), source.New(input, "RETURN 1"), output); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	writeQuery(t, input, "RETURN 2")

	restore := stubRenameArtifactFile(func(string, string) error {
		return errors.New("rename failed")
	})
	defer restore()

	err := WriteArtifact(t.Context(), newCompiler(t), source.New(input, "RETURN 2"), output)

	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "replace "+output+" with temporary artifact") {
		t.Fatalf("unexpected error: %v", err)
	}

	assertArtifactSource(t, output, "RETURN 1")
	assertNoTempArtifacts(t, filepath.Dir(output), output)
}

func TestWriteArtifact_RenameFailureDoesNotCreateDestinationAndCleansTemp(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "query.fql")
	output := filepath.Join(dir, "query.fqlc")

	writeQuery(t, input, "RETURN 42")

	restore := stubRenameArtifactFile(func(string, string) error {
		return errors.New("rename failed")
	})
	defer restore()

	err := WriteArtifact(t.Context(), newCompiler(t), source.New(input, "RETURN 42"), output)

	if err == nil {
		t.Fatal("expected error")
	}

	if _, statErr := os.Stat(output); !os.IsNotExist(statErr) {
		t.Fatalf("expected artifact to be absent, stat err=%v", statErr)
	}

	assertNoTempArtifacts(t, filepath.Dir(output), output)
}

func TestWriteArtifact_ArtifactRoundTrip(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "query.fql")
	output := filepath.Join(dir, "query.fqlc")

	writeQuery(t, input, "RETURN 42")

	if err := WriteArtifact(t.Context(), newCompiler(t), source.New(input, "RETURN 42"), output); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertArtifactSource(t, output, "RETURN 42")
}

func TestWriteArtifact_InvalidQueryDoesNotCreateArtifactInMissingParentDirectory(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "broken.fql")
	output := filepath.Join(dir, "nested", "out", "broken.fqlc")

	writeQuery(t, input, "FOR item IN")

	err := WriteArtifact(t.Context(), newCompiler(t), source.New(input, "FOR item IN"), output)

	if err == nil {
		t.Fatal("expected error")
	}

	if _, statErr := os.Stat(output); !os.IsNotExist(statErr) {
		t.Fatalf("expected artifact to be absent, stat err=%v", statErr)
	}
}

func TestWriteArtifact_CanceledContextLeavesDestinationUntouched(t *testing.T) {
	tests := []struct {
		name          string
		missingParent bool
		existing      bool
	}{
		{name: "missing destination"},
		{name: "missing parent", missingParent: true},
		{name: "existing destination", existing: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			input := filepath.Join(dir, "query.fql")
			output := filepath.Join(dir, "dist", "query.fqlc")
			c := newCompiler(t)

			writeQuery(t, input, "RETURN 2")

			if !test.missingParent {
				if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
					t.Fatal(err)
				}
			}

			var original []byte

			if test.existing {
				if err := WriteArtifact(t.Context(), c, source.New(input, "RETURN 1"), output); err != nil {
					t.Fatal(err)
				}

				var err error
				original, err = os.ReadFile(output)
				if err != nil {
					t.Fatal(err)
				}
			}

			ctx, cancel := context.WithCancel(t.Context())
			cancel()

			err := WriteArtifact(ctx, c, source.New(input, "RETURN 2"), output)
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("expected cancellation, got %v", err)
			}

			if test.existing {
				content, err := os.ReadFile(output)
				if err != nil {
					t.Fatal(err)
				}

				if !bytes.Equal(content, original) {
					t.Fatal("canceled compilation replaced the existing artifact")
				}
			} else if _, err := os.Stat(output); !os.IsNotExist(err) {
				t.Fatalf("expected artifact to be absent, got %v", err)
			}

			if test.missingParent {
				if _, err := os.Stat(filepath.Dir(output)); !os.IsNotExist(err) {
					t.Fatalf("expected parent directory to be absent, got %v", err)
				}
			}

			assertNoTempArtifacts(t, filepath.Dir(output), output)
		})
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

	if program.Source.Empty() {
		t.Fatal("expected serialized source")
	}

	if program.Source.Content() != expected {
		t.Fatalf("expected source %q, got %q", expected, program.Source.Content())
	}
}

func newCompiler(t *testing.T) *compiler.Compiler {
	t.Helper()

	c, err := compiler.New()
	if err != nil {
		t.Fatal(err)
	}

	return c
}

func assertNoTempArtifacts(t *testing.T, dir, outputPath string) {
	t.Helper()

	matches, err := filepath.Glob(filepath.Join(dir, artifactTempPattern(outputPath)))
	if err != nil {
		t.Fatal(err)
	}

	if len(matches) != 0 {
		t.Fatalf("expected temporary artifacts to be cleaned up, got %v", matches)
	}
}

func stubRenameArtifactFile(fn func(string, string) error) func() {
	previous := renameArtifactFile
	renameArtifactFile = fn

	return func() {
		renameArtifactFile = previous
	}
}
