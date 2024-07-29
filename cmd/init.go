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

	// create and switch to FileTreeBranch and defer switching back to original branch
	originalBranch, err := core.SwitchToFileTreeBranch()
	if err != nil {
		fmt.Println("failed to switch to filetree branch")
		return err
	}
	defer core.SwitchToBranch(originalBranch)
	defer core.StashPop()

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

	// create .gitattributes file
	// if err := core.CreateGitAttributes(); err != nil {
	// 	fmt.Println("failed to create .gitattributes")
	// 	return err
	// }

	// stage and commit filetree.yaml to FileTreeBranch
	// if err := core.StageAndCommit(".", "Initialize filetree.yaml"); err != nil {
	// 	fmt.Println("failed to stage and commit filetree.yaml")
	// 	return err
	// }

	return nil
}
