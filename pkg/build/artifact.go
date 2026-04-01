package build

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/MontFerret/ferret/v2/pkg/bytecode/artifact"
	"github.com/MontFerret/ferret/v2/pkg/compiler"
	"github.com/MontFerret/ferret/v2/pkg/source"
)

var renameArtifactFile = os.Rename

func WriteArtifact(c *compiler.Compiler, src *source.Source, outputPath string) error {
	same, err := samePath(src.Name(), outputPath)

	if err != nil {
		return err
	}

	if same {
		return fmt.Errorf("output path %s would overwrite source file %s", outputPath, src.Name())
	}

	program, err := c.Compile(src)

	if err != nil {
		return err
	}

	data, err := artifact.Marshal(program, artifact.Options{})

	if err != nil {
		return fmt.Errorf("serialize %s: %w", src.Name(), err)
	}

	outputDir := filepath.Dir(outputPath)

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("create output directory %s: %w", outputDir, err)
	}

	tempFile, err := os.CreateTemp(outputDir, artifactTempPattern(outputPath))

	if err != nil {
		return fmt.Errorf("create temporary artifact for %s: %w", outputPath, err)
	}

	tempPath := tempFile.Name()
	cleanupTemp := true
	defer func() {
		if !cleanupTemp {
			return
		}

		_ = tempFile.Close()
		_ = os.Remove(tempPath)
	}()

	if _, err := tempFile.Write(data); err != nil {
		return fmt.Errorf("write temporary artifact for %s: %w", outputPath, err)
	}

	if err := tempFile.Chmod(0o644); err != nil {
		return fmt.Errorf("set permissions on temporary artifact for %s: %w", outputPath, err)
	}

	if err := tempFile.Sync(); err != nil {
		return fmt.Errorf("sync temporary artifact for %s: %w", outputPath, err)
	}

	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("close temporary artifact for %s: %w", outputPath, err)
	}

	if err := renameArtifactFile(tempPath, outputPath); err != nil {
		return fmt.Errorf("replace %s with temporary artifact: %w", outputPath, err)
	}

	cleanupTemp = false

	return nil
}

func artifactTempPattern(outputPath string) string {
	return "." + filepath.Base(outputPath) + ".tmp-*"
}
