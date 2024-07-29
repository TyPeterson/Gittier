package cmd

import (
	"fmt"

	"github.com/TyPeterson/Gittier/core"
)

func Commit() error {

	// sync the file tree first
	if err := Sync(); err != nil {
		return fmt.Errorf("failed to sync: %w", err)
	}

	// switch to FileTreeBranch branch
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

	// read filetree.yaml into an in-memory FileTree
	fileTree, err := core.ReadFileTreeFromYaml("filetree.yaml")
	if err != nil {
		return fmt.Errorf("failed to read filetree.yaml: %w", err)
	}

	orderedNodes := core.GetDfsOrder(fileTree)

	for _, node := range orderedNodes {
		if node.IsDir {
			if err := core.CommitFolder(node); err != nil {
				return fmt.Errorf("failed to commit folder: %w", err)
			}
		} else {
			if err := core.CommitFile(node); err != nil {
				return fmt.Errorf("failed to commit file: %w", err)
			}
		}
	}

	// TODO: add final commit that is shown at top of repo next to username

	fmt.Println("All files and folders committed successfully")
	return nil
}
