package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/TyPeterson/Gittier/core"
)

func Desc(path string, description string, verbose bool) error {
	// Normalize the path
	path = filepath.Clean(path)

	// Read the existing FileTree
	fileTree, err := core.ReadFileTreeFromYaml("filetree.yaml")
	if err != nil {
		return fmt.Errorf("failed to read filetree.yaml: %w", err)
	}

	// Check if the path exists in the FileTree
	node, exists := fileTree.Nodes[path]
	if !exists {
		return fmt.Errorf("path not found in filetree: %s", path)
	}

	// In verbose mode, show the old description
	if verbose && node.Description != "" {
		fmt.Printf("Current description for '%s': %s\n", path, node.Description)
	}

	// If the node already has a description, ask for confirmation
	if node.Description != "" {
		fmt.Printf("Do you want to overwrite the existing description? (y/n): ")
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" {
			fmt.Println("Operation cancelled.")
			return nil
		}
	}

	// Update the description
	node.Description = description

	// Write the updated FileTree back to filetree.yaml
	if err := core.WriteFileTreeToYaml(fileTree, "filetree.yaml"); err != nil {
		return fmt.Errorf("failed to write updated filetree.yaml: %w", err)
	}

	fmt.Printf("Updated description for '%s'\n", path)
	return nil
}

// Temporary line for commit
