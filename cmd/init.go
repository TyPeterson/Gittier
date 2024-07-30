package cmd

import (
	"errors"
	"fmt"

	"github.com/TyPeterson/Gittier/core"
)

// ---------- cmdInit ----------
func Init() error {
	// ensure the current directory is a git repo
	if !core.IsGitRepo() {
		return errors.New("Not a git repository")
	}

	// ensure the project is not already initialized
	if core.BranchExists(core.FileTreeBranch) {
		return errors.New("Project already initialized, run 'gittier update' instead")
	}

	// save the current branch name
	currentBranch, err := core.GetCurrentBranch()

	if err != nil {
		fmt.Println("failed to get current branch")
		return err
	}

	// create FileTreeBranch
	if err := core.CreateBranch(core.FileTreeBranch); err != nil {
		return fmt.Errorf("failed to create filetree branch: %w", err)
	}

	// stash any changes if needed
	needToStash, err := core.NeedToStash(currentBranch)
	if err != nil {
		fmt.Println("failed to check if need to stash")
		return err
	}

	if needToStash {
		if err := core.Stash(); err != nil {
			fmt.Println("failed to stash")
			return err
		}

		defer func() {
			if err := core.StashPop(); err != nil {
				fmt.Println("failed to pop stash")
			}

		}()
	}

	// switch to FileTreeBranch
	if err := core.SwitchToBranch(core.FileTreeBranch); err != nil {
		fmt.Println("failed to switch to filetree branch")
		return err
	}

	// defer switching back to current branch
	defer func() {
		if err := core.SwitchToBranch(currentBranch); err != nil {
			fmt.Println("failed to switch back to current branch")
		}
	}()

	// get FileTree from main branch's ls-tree
	fileTree, err := core.GetFileTreeFromBranch("main")
	if err != nil {
		fmt.Println("failed to get file tree from ls-tree")
		return err
	}

	// write FileTree to filetree.yaml
	if err := core.WriteFileTreeToYaml(fileTree, "filetree.yaml"); err != nil {
		fmt.Println("failed to write filetree.yaml")
		return err
	}

	// stage and commit filetree.yaml
	if err := core.StageAndCommit("filetree.yaml", "Initialize filetree.yaml"); err != nil {
		fmt.Println("failed to stage and commit filetree.yaml and .gitattributes")
		return err
	}

	fmt.Println("Gittier project initialized")
	return nil
}
