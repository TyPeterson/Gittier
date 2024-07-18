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

	// check if filetree.yaml already exists
	if core.FileExists("filetree.yaml") {
		if !core.ConfirmOverwrite("filetree.yaml") {
			return errors.New("Aborted")
		}
	}

	// add or update .gitignore
	if err := core.AddToGitignore("filetree.yaml"); err != nil {
		return err
	}

	// get current commit hash
	commitHash, err := core.GetCurrentCommitHash()
	if err != nil {
		return err
	}

	// get FileTree from ls-tree
	fileTree, err := core.GetFileTreeFromLsTree()
	if err != nil {
		return err
	}

	fileTree.CommitHash = commitHash

	// write FileTree to filetree.yaml
	if err := core.WriteFileTreeToYaml(fileTree, "filetree.yaml"); err != nil {
		return err
	}

	fmt.Println("filetree.yaml initialized successfully")

	// fmt.Println("filetree contents in dfs order:")
	// orderedNodes := core.GetDfsOrder(fileTree)
	// if len(orderedNodes) == 0 {
	// 	fmt.Println("empty")
	// }
	// for _, node := range orderedNodes {
	// 	fmt.Println(node.Path)
	// }

	return nil
}
