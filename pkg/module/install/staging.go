package install

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type installStage struct {
	directory     string
	modPath       string
	sumPath       string
	sourcePath    string
	overlayPath   string
	buildOutput   string
	goModSnapshot fileSnapshot
	goSumSnapshot fileSnapshot
}

func newInstallStage(project *projectInfo) (*installStage, error) {
	goModSnapshot, err := snapshotFile(project.GoModPath)
	if err != nil {
		return nil, err
	}

	goSumSnapshot, err := snapshotFile(project.GoSumPath)
	if err != nil {
		return nil, err
	}

	directory, err := os.MkdirTemp("", "ferret-mod-install-*")
	if err != nil {
		return nil, fmt.Errorf("create module installation staging directory: %w", err)
	}

	stage := &installStage{
		directory:     directory,
		modPath:       filepath.Join(directory, "project.mod"),
		sumPath:       filepath.Join(directory, "project.sum"),
		overlayPath:   filepath.Join(directory, "overlay.json"),
		buildOutput:   filepath.Join(directory, "package-build"),
		goModSnapshot: goModSnapshot,
		goSumSnapshot: goSumSnapshot,
	}

	if err := os.WriteFile(stage.modPath, goModSnapshot.Data, goModSnapshot.Mode.Perm()); err != nil {
		os.RemoveAll(directory)
		return nil, fmt.Errorf("stage go.mod: %w", err)
	}

	if goSumSnapshot.Exists {
		if err := os.WriteFile(stage.sumPath, goSumSnapshot.Data, goSumSnapshot.Mode.Perm()); err != nil {
			os.RemoveAll(directory)
			return nil, fmt.Errorf("stage go.sum: %w", err)
		}
	}

	return stage, nil
}

func stageInstallComposition(stage *installStage, target *composition, source []byte) (fileSnapshot, error) {
	snapshot, err := snapshotFile(target.Filename)
	if err != nil {
		return fileSnapshot{}, err
	}

	stage.sourcePath = filepath.Join(stage.directory, filepath.Base(target.Filename))
	if err := os.WriteFile(stage.sourcePath, source, snapshot.Mode.Perm()); err != nil {
		return fileSnapshot{}, fmt.Errorf("stage composition source: %w", err)
	}

	overlay, err := json.Marshal(overlayDocument{Replace: map[string]string{target.Filename: stage.sourcePath}})
	if err != nil {
		return fileSnapshot{}, fmt.Errorf("encode Go source overlay: %w", err)
	}

	if err := os.WriteFile(stage.overlayPath, overlay, 0o600); err != nil {
		return fileSnapshot{}, fmt.Errorf("stage Go source overlay: %w", err)
	}

	return snapshot, nil
}
