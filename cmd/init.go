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

	// if FileTreeBranch exists, return error message
	if core.BranchExists(core.FileTreeBranch) {
		return errors.New("Project already initialized, run 'gittier update' instead")
	}

	// create FileTreeBranch
	currentBranch, err := core.GetCurrentBranch()
	if err != nil {
		fmt.Println("failed to get current branch")
		return err
	}

	if err := core.CreateFileTreeBranch(); err != nil {
		fmt.Println("failed to create filetree branch")
		return err
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

	// get current commit hash
	commitHash, err := core.GetCurrentCommitHash()
	if err != nil {
		fmt.Println("failed to get current commit hash")
		return err
	}

	// get FileTree from ls-tree
	fileTree, err := core.GetFileTreeFromLsTree()
	if err != nil {
		fmt.Println("failed to get file tree from ls-tree")
		return err
	}

	fileTree.CommitHash = commitHash

	// write FileTree to filetree.yaml
	if err := core.WriteFileTreeToYaml(fileTree, "filetree.yaml"); err != nil {
		fmt.Println("failed to write filetree.yaml")
		return err
	}

	// write filetree.yaml to main branch's gitignore
	if err := core.AddToGitignore("filetree.yaml"); err != nil {
		fmt.Println("failed to add filetree.yaml to .gitignore")
		return err
	}

	// create .gitattributes file
	if err := core.CreateGitAttributes(); err != nil {
		fmt.Println("failed to create .gitattributes")
		return err
	}

	// stage and commit filetree.yaml and .gitattributes to FileTreeBranch
	if err := core.StageAndCommit(".", "Initialize filetree.yaml and .gitattributes"); err != nil {
		fmt.Println("failed to stage and commit filetree.yaml and .gitattributes")
		return err
	}

	fmt.Println("Project initialized")
	return nil
}
