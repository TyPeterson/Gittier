package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/TyPeterson/Gittier/core"
)

func Desc(path string, description string, verbose bool) error {
	path = filepath.Clean(path)

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

	// read the existing FileTree into an in-memory representation
	fileTree, err := core.ReadFileTreeFromYaml("filetree.yaml")
	if err != nil {
		return fmt.Errorf("failed to read filetree.yaml: %w", err)
	}

	node, exists := fileTree.Nodes[path]
	if !exists {
		return fmt.Errorf("path not found in filetree: %s", path)
	}

	// in verbose mode, show the old description
	if verbose && node.Description != "" {
		fmt.Printf("Current description for '%s': %s\n", path, node.Description)
	}

	// if the node already has a description, ask for confirmation
	if node.Description != "" {
		fmt.Printf("Do you want to overwrite the existing description? (y/n): ")
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" {
			fmt.Println("Operation cancelled.")
			return nil
		}
	}

	node.Description = description

	// write the updated FileTree back to filetree.yaml
	if err := core.WriteFileTreeToYaml(fileTree, "filetree.yaml"); err != nil {
		return fmt.Errorf("failed to write updated filetree.yaml: %w", err)
	}

	fmt.Printf("Updated description for '%s'\n", path)

	// stage and commit filetree.yaml to FileTreeBranch
	if err := core.StageAndCommit("filetree.yaml", "Initialize filetree.yaml"); err != nil {
		fmt.Println("failed to stage and commit filetree.yaml")
		return err
	}

	return nil
}
