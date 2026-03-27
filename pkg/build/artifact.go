package build

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/MontFerret/ferret/v2/pkg/bytecode/artifact"
	"github.com/MontFerret/ferret/v2/pkg/compiler"
	"github.com/MontFerret/ferret/v2/pkg/file"
)

func WriteArtifact(c *compiler.Compiler, src *file.Source, outputPath string) error {
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

	if err := os.WriteFile(outputPath, data, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", outputPath, err)
	}

	return nil
}
