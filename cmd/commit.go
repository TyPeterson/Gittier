package cmd

import (
	"fmt"

	"github.com/TyPeterson/Gittier/core"
)

func Commit() error {
	fileTree, err := core.ReadFileTreeFromYaml("filetree.yaml")
	if err != nil {
		return fmt.Errorf("failed to read filetree.yaml: %w", err)
	}

	nodes := core.GetDfsOrder(fileTree)

	for _, node := range nodes {
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
