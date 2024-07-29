package cmd

import (
	"fmt"

	"github.com/TyPeterson/Gittier/core"
)

func Sync() error {

	currentBranch, err := core.GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	needToStash, err := core.NeedToStash(currentBranch)
	if err != nil {
		return fmt.Errorf("failed to check if need to stash: %w", err)
	}

	if needToStash {
		if err := core.Stash(); err != nil {
			return fmt.Errorf("failed to stash: %w", err)
		}

		defer func() {
			if err := core.StashPop(); err != nil {
				fmt.Println("failed to pop stash")
			}
		}()
	}

	if err := core.SwitchToBranch(core.FileTreeBranch); err != nil {
		return fmt.Errorf("failed to switch to filetree branch: %w", err)
	}

	defer func() {
		if err := core.SwitchToBranch(currentBranch); err != nil {
			fmt.Println("failed to switch to original branch")
		}
	}()

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
	// if diffOutput is empty, no changes have been made to the file tree and we can return
	if len(diffOutput) == 0 || (len(diffOutput) == 1 && diffOutput[0] == "") {
		fmt.Println("No changes to sync")
		return nil
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
