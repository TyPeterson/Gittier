package cmd

import (
	"fmt"

	"github.com/TyPeterson/Gittier/core"
)

func Sync() error {
	// switch to FileTreeBranch, create if it doesn't exist, and defer switching back to original branch
	originalBranch, err := core.SwitchToFileTreeBranch()
	if err != nil {
		fmt.Println("failed to switch to filetree branch")
		return err
	}
	defer core.SwitchToBranch(originalBranch)
	defer core.StashPop()

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

	// apply changes from Sync to oldFileTree
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

	// stage and commit filetree.yaml to FileTreeBranch
	if err := core.StageAndCommit("filetree.yaml", "Initialize filetree.yaml"); err != nil {
		fmt.Println("failed to stage and commit filetree.yaml")
		return err
	}

	fmt.Println("File tree updated")
	return nil
}
