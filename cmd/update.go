package cmd

import (
	"fmt"

	"github.com/TyPeterson/Gittier/core"
)

func Update() error {
	// switch to FileTreeBranch and defer back to the original branch
	originalBranch, err := core.SwitchToFileTreeBranch()
	if err != nil {
		return fmt.Errorf("failed to switch to filetree branch: %w", err)
	}
	defer core.SwitchToBranch(originalBranch)

	// read filetree.yaml into a FileTree
	oldFileTree, err := core.ReadFileTreeFromYaml("filetree.yaml")
	if err != nil {
		return fmt.Errorf("failed to read filetree.yaml: %w", err)
	}

	// get diff between commit hash of filetree.yaml and main
	diffOutput, err := core.GetDiffOutput(oldFileTree.CommitHash)
	if err != nil {
		return fmt.Errorf("failed to get diff output: %w", err)
	}

	// apply changes from diffOutput to oldFileTree
	updatedFileTree, err := core.ProcessGitDiff(oldFileTree, diffOutput)
	if err != nil {
		return fmt.Errorf("failed to process git diff: %w", err)
	}

	// get current structure of main branch FileTree
	currentFileTree, err := core.GetFileTreeFromLsTree()
	if err != nil {
		return fmt.Errorf("failed to get file tree from ls-tree: %w", err)
	}

	// arrange the updatedFileTree to match the DFS order of the currentFileTree before writing to yaml
	syncedFileTree := core.SyncFileTree(updatedFileTree, currentFileTree)

	if err := core.WriteFileTreeToYaml(syncedFileTree, "filetree.yaml"); err != nil {
		return fmt.Errorf("failed to write file tree to yaml: %w", err)
	}

	fmt.Println("File tree updated")
	return nil
}
