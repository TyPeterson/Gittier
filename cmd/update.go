package cmd

import (
	// "bufio"
	"fmt"
	// "os/exec"
	// "strings"

	"github.com/TyPeterson/Gittier/core"
)

func Update() error {
	oldFileTree, err := core.ReadFileTreeFromYaml("filetree.yaml")
	if err != nil {
		return fmt.Errorf("failed to read filetree.yaml: %w", err)
	}

	diffOutput, err := core.GetDiffOutput(oldFileTree.CommitHash)
	if err != nil {
		return fmt.Errorf("failed to get diff output: %w", err)
	}

	updatedFileTree, err := core.ProcessGitDiff(oldFileTree, diffOutput)
	if err != nil {
		return fmt.Errorf("failed to process git diff: %w", err)
	}

	currentFileTree, err := core.GetFileTreeFromLsTree()
	if err != nil {
		return fmt.Errorf("failed to get file tree from ls-tree: %w", err)
	}

	syncedFileTree := core.SyncFileTree(updatedFileTree, currentFileTree)

	if err := core.WriteFileTreeToYaml(syncedFileTree, "filetree.yaml"); err != nil {
		return fmt.Errorf("failed to write file tree to yaml: %w", err)
	}

	fmt.Println("File tree updated")
	return nil
}
