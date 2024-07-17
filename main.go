package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	gitignore "github.com/sabhiram/go-gitignore"
	"gopkg.in/yaml.v2"
)

func stageAndCommitFile(path string, description string) error {
	// Step 1: Add a temporary line
	err := addTemporaryLine(path)
	if err != nil {
		return fmt.Errorf("failed to add temporary line: %w", err)
	}

	// Commit the temporary change
	_, err = executeGitCommand("add", path)
	if err != nil {
		return fmt.Errorf("failed to stage temporary change: %w", err)
	}

	_, err = executeGitCommand("commit", "-m", fmt.Sprintf("%s temp commit", filepath.Base(path)))
	if err != nil {
		return fmt.Errorf("failed to commit temporary change: %w", err)
	}

	// Step 2: Remove the temporary line
	err = removeTemporaryLine(path)
	if err != nil {
		return fmt.Errorf("failed to remove temporary line: %w", err)
	}

	// Commit the removal with the actual description
	_, err = executeGitCommand("add", path)
	if err != nil {
		return fmt.Errorf("failed to stage file: %w", err)
	}

	_, err = executeGitCommand("commit", "-m", description)
	if err != nil {
		return fmt.Errorf("failed to commit file with description: %w", err)
	}

	return nil
}

func addTemporaryLine(path string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString("\n// Temporary line for commit\n")
	return err
}

func removeTemporaryLine(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	if len(lines) < 2 {
		return nil // File is too short, do nothing
	}

	// Remove the last two lines (the temporary line and the newline before it)
	newContent := strings.Join(lines[:len(lines)-2], "\n")

	return os.WriteFile(path, []byte(newContent), 0644)
}

func executeGitCommand(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git command failed: %s, %w", string(output), err)
	}
	return string(output), nil
}

func commitAllFiles(nodes []FileNode) error {
	for _, node := range nodes {
		if !node.IsDir {
			err := stageAndCommitFile(node.Path, node.Description)
			if err != nil {
				return fmt.Errorf("failed to commit file %s: %w", node.Path, err)
			}
		}
	}
	return nil
}

// ----------------- FileNode -----------------
type FileNode struct {
	Path        string `yaml:"path"`
	Description string `yaml:"description"`
	IsDir       bool   `yaml:"is_dir"`
}

func scanDirectory(root string) ([]FileNode, error) {
	var nodes []FileNode

	ignoreFilePath := filepath.Join(root, ".gitignore")
	var ignore *gitignore.GitIgnore
	if _, err := os.Stat(ignoreFilePath); err == nil {
		ignore, err = gitignore.CompileIgnoreFile(ignoreFilePath)
		if err != nil {
			return nil, fmt.Errorf("error compiling .gitignore: %w", err)
		}
	}

	var dfs func(path string) error
	dfs = func(path string) error {
		entries, err := os.ReadDir(path)
		if err != nil {
			return err
		}

		// Sort entries to ensure consistent ordering
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Name() < entries[j].Name()
		})

		for _, entry := range entries {
			fullPath := filepath.Join(path, entry.Name())
			relPath, err := filepath.Rel(root, fullPath)
			if err != nil {
				return err
			}

			// Skip .git directory
			cleanPath := filepath.Clean(relPath)
			if cleanPath == ".git" || (cleanPath != ".gitignore" && strings.HasPrefix(cleanPath, ".git")) {
				continue
			}

			// Check if file is ignored by .gitignore
			if ignore != nil && ignore.MatchesPath(relPath) {
				continue
			}

			info, err := entry.Info()
			if err != nil {
				return err
			}

			if info.IsDir() {
				// Recursively process subdirectories
				if err := dfs(fullPath); err != nil {
					return err
				}
			}

			// Add the node after processing its children (if it's a directory)
			node := FileNode{
				Path:        relPath,
				Description: "No description added",
				IsDir:       info.IsDir(),
			}
			nodes = append(nodes, node)
		}
		return nil
	}

	if err := dfs(root); err != nil {
		return nil, err
	}

	return nodes, nil
}

func saveYAML(nodes []FileNode, filename string) error {
	data, err := yaml.Marshal(nodes)
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}

func loadYAML(filename string) ([]FileNode, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var nodes []FileNode
	err = yaml.Unmarshal(data, &nodes)
	if err != nil {
		return nil, err
	}

	return nodes, nil
}

func printDFS(nodes []FileNode) {

	for _, node := range nodes {
		fmt.Println(node.Path, ":", node.Description)
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <command>")
		return
	}

	command := os.Args[1]

	switch command {
	case "init":
		root, err := os.Getwd()
		if err != nil {
			fmt.Printf("Error getting current directory: %v\n", err)
			return
		}

		nodes, err := scanDirectory(root)
		if err != nil {
			fmt.Printf("Error scanning directory: %v\n", err)
			return
		}

		err = saveYAML(nodes, "filetree.yaml")
		if err != nil {
			fmt.Printf("Error saving filetree: %v\n", err)
			return
		}

		fmt.Println("Filetree initialized")

	case "dfs":
		nodes, err := loadYAML("filetree.yaml")
		if err != nil {
			fmt.Printf("Error loading filetree: %v\n", err)
			return
		}

		fmt.Println("Depth-First Search Traversal:")
		printDFS(nodes)

	case "commit-all":
		nodes, err := loadYAML("filetree.yaml")
		if err != nil {
			fmt.Printf("Error loading filetree: %v\n", err)
			return
		}

		err = commitAllFiles(nodes)
		if err != nil {
			fmt.Printf("Error committing files: %v\n", err)
			return
		}

		fmt.Println("All files have been committed with their descriptions")
	default:
		fmt.Println("Invalid command")
	}
}
