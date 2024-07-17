package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"time"

	// "golang.org/x/sys/unix"

	gitignore "github.com/sabhiram/go-gitignore"
	"gopkg.in/yaml.v2"
)

func modifyFileTimestamp(path string) error {
	// Get current file info
	_, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	// Get current time
	now := time.Now().Local()

	// Change both the access time and modification time
	err = os.Chtimes(path, now, now)
	if err != nil {
		return fmt.Errorf("failed to change file timestamps: %w", err)
	}

	return nil
}

func stageFileForGit(path string) error {
	// First, modify the file timestamp
	err := modifyFileTimestamp(path)
	if err != nil {
		return err
	}

	// Then, stage the file using Git
	_, err = executeGitCommand("add", path)
	if err != nil {
		return fmt.Errorf("failed to stage file: %w", err)
	}

	return nil
}

func executeGitCommand(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git command failed: %s, %w", string(output), err)
	}
	return string(output), nil
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

	case "stage":
		if len(os.Args) < 3 {
			fmt.Println("Usage: go run main.go stage <file_path>")
			return
		}
		filePath := os.Args[2]
		err := stageFileForGit(filePath)
		if err != nil {
			fmt.Printf("Error staging file: %v\n", err)
			return
		}
		fmt.Printf("File %s has been staged for commit\n", filePath)
	default:
		fmt.Println("Invalid command")
	}
}
