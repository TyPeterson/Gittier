package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	gitignore "github.com/sabhiram/go-gitignore"
	"gopkg.in/yaml.v2"
)

type FileTree struct {
	CommitHash string     `yaml:"commit_hash"`
	Nodes      []FileNode `yaml:"nodes"`
}

type FileNode struct {
	Path        string `yaml:"path"`
	Description string `yaml:"description"`
	IsDir       bool   `yaml:"is_dir"`
}

func getCurrentCommitHash() (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("error getting current commit hash: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

func scanDirectory(root string) ([]FileNode, error) {
	var nodes []FileNode

	ignoreFilePath := filepath.Join(root, ".gitignore")
	var ignore *gitignore.GitIgnore
	var err error
	if _, err := os.Stat(ignoreFilePath); err == nil {
		ignore, err = gitignore.CompileIgnoreFile(ignoreFilePath)
		if err != nil {
			return nil, fmt.Errorf("error compiling .gitignore: %w", err)
		}
	}

	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		if relPath == "." {
			return nil
		}

		if relPath == ".git" || strings.HasPrefix(relPath, ".git"+string(os.PathSeparator)) {
			return filepath.SkipDir
		}

		if ignore != nil && ignore.MatchesPath(relPath) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		node := FileNode{
			Path:        relPath,
			Description: "No description added",
			IsDir:       info.IsDir(),
		}
		nodes = append(nodes, node)

		return nil
	})

	if err != nil {
		return nil, err
	}

	return nodes, nil
}

func getGitChanges(oldCommit string) (map[string]string, error) {
	cmd := exec.Command("git", "diff", "--name-status", oldCommit, "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("error running git diff: %w", err)
	}

	changes := make(map[string]string)
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		parts := strings.Split(strings.TrimSpace(line), "\t")
		if len(parts) >= 2 {
			status := parts[0]
			path := parts[1]
			changes[path] = status
		}
	}

	return changes, nil
}

func syncFileTree(oldTree FileTree, newNodes []FileNode) (FileTree, error) {
	changes, err := getGitChanges(oldTree.CommitHash)
	if err != nil {
		return FileTree{}, err
	}

	oldMap := make(map[string]FileNode)
	for _, node := range oldTree.Nodes {
		oldMap[node.Path] = node
	}

	var syncedNodes []FileNode
	for _, newNode := range newNodes {
		if status, changed := changes[newNode.Path]; changed {
			switch status[0] {
			case 'A':
				syncedNodes = append(syncedNodes, newNode)
			case 'M':
				if oldNode, exists := oldMap[newNode.Path]; exists {
					newNode.Description = oldNode.Description
				}
				syncedNodes = append(syncedNodes, newNode)
			case 'R':
				for oldPath, oldNode := range oldMap {
					if changes[oldPath] == "R"+newNode.Path {
						newNode.Description = oldNode.Description
						break
					}
				}
				syncedNodes = append(syncedNodes, newNode)
			}
		} else if _, exists := oldMap[newNode.Path]; exists {
			newNode.Description = oldMap[newNode.Path].Description
			syncedNodes = append(syncedNodes, newNode)
		} else {
			syncedNodes = append(syncedNodes, newNode)
		}
	}

	newCommitHash, err := getCurrentCommitHash()
	if err != nil {
		return FileTree{}, err
	}

	return FileTree{
		CommitHash: newCommitHash,
		Nodes:      syncedNodes,
	}, nil
}

func saveYAML(tree FileTree, filename string) error {
	data, err := yaml.Marshal(tree)
	if err != nil {
		return err
	}

	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return err
	}

	// Ensure .gitignore exists and contains the filename
	gitignorePath := ".gitignore"
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		// Create .gitignore if it doesn't exist
		_, err = os.Create(gitignorePath)
		if err != nil {
			return fmt.Errorf("failed to create .gitignore: %w", err)
		}
	}

	// Read .gitignore
	ignoreFile, err := os.OpenFile(gitignorePath, os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open .gitignore: %w", err)
	}
	defer ignoreFile.Close()

	scanner := bufio.NewScanner(ignoreFile)
	var found bool
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) == filename {
			found = true
			break
		}
	}

	// Append filename to .gitignore if not found
	if !found {
		_, err = ignoreFile.WriteString(filename + "\n")
		if err != nil {
			return fmt.Errorf("failed to update .gitignore: %w", err)
		}
	}

	return nil
}

func loadYAML(filename string) (FileTree, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return FileTree{}, err
	}

	var tree FileTree
	err = yaml.Unmarshal(data, &tree)
	if err != nil {
		return FileTree{}, err
	}

	return tree, nil
}

func commitFile(filePath string, description string) error {
	// Check if the file is ignored by Git
	cmd := exec.Command("git", "check-ignore", "-q", filePath)
	if err := cmd.Run(); err == nil {
		// File is ignored, skip it
		fmt.Printf("Skipping ignored file: %s\n", filePath)
		return nil
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	tempLine := fmt.Sprintf("\nTemporary line added at %s\n", time.Now().Format(time.RFC3339))
	newContent := append(content, []byte(tempLine)...)

	if err := os.WriteFile(filePath, newContent, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	cmd = exec.Command("git", "add", filePath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git add failed: %w", err)
	}

	cmd = exec.Command("git", "commit", "-m", fmt.Sprintf("%s temp change", filepath.Base(filePath)))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git commit (temp) failed: %w", err)
	}

	if err = os.WriteFile(filePath, content, 0644); err != nil {
		return fmt.Errorf("failed to restore original content: %w", err)
	}

	cmd = exec.Command("git", "add", filePath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git add (final) failed: %w", err)
	}

	cmd = exec.Command("git", "commit", "-m", description)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git commit (final) failed: %w", err)
	}

	return nil
}

func commitDirectory(dirPath string, description string) error {
	// Check if the directory is ignored by Git
	cmd := exec.Command("git", "check-ignore", "-q", dirPath)
	if err := cmd.Run(); err == nil {
		// Directory is ignored, skip it
		fmt.Printf("Skipping ignored directory: %s\n", dirPath)
		return nil
	}

	tempFileName := fmt.Sprintf("temp_file_%d.txt", time.Now().UnixNano())
	tempFilePath := filepath.Join(dirPath, tempFileName)
	tempFile, err := os.Create(tempFilePath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tempFile.WriteString("Temporary file for commit")
	tempFile.Close()

	cmd = exec.Command("git", "add", tempFilePath)
	if err := cmd.Run(); err != nil {
		os.Remove(tempFilePath)
		return fmt.Errorf("git add (temp file) failed: %w", err)
	}

	cmd = exec.Command("git", "commit", "-m", fmt.Sprintf("Add temporary file in %s", dirPath))
	if err := cmd.Run(); err != nil {
		os.Remove(tempFilePath)
		return fmt.Errorf("git commit (temp file) failed: %w", err)
	}

	if err := os.Remove(tempFilePath); err != nil {
		return fmt.Errorf("failed to remove temp file: %w", err)
	}

	cmd = exec.Command("git", "add", tempFilePath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git add (remove temp file) failed: %w", err)
	}

	cmd = exec.Command("git", "commit", "-m", description)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git commit (final) failed: %w", err)
	}

	return nil
}

func commitNode(node FileNode) error {
	if node.IsDir {
		return commitDirectory(node.Path, node.Description)
	}
	return commitFile(node.Path, node.Description)
}

func commitAll(nodes []FileNode) error {
	for _, node := range nodes {
		fmt.Printf("Committing: %s\n", node.Path)
		if err := commitNode(node); err != nil {
			return fmt.Errorf("error committing %s: %w", node.Path, err)
		}
	}
	return nil
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

		commitHash, err := getCurrentCommitHash()
		if err != nil {
			fmt.Printf("Error getting current commit hash: %v\n", err)
			return
		}

		tree := FileTree{
			CommitHash: commitHash,
			Nodes:      nodes,
		}

		err = saveYAML(tree, "filetree.yaml")
		if err != nil {
			fmt.Printf("Error saving filetree: %v\n", err)
			return
		}

		fmt.Println("Filetree initialized")

	case "commit":
		if len(os.Args) < 3 {
			fmt.Println("Usage: go run main.go commit <path>")
			return
		}
		path := os.Args[2]

		oldTree, err := loadYAML("filetree.yaml")
		if err != nil {
			fmt.Printf("Error loading filetree: %v\n", err)
			return
		}

		root, err := os.Getwd()
		if err != nil {
			fmt.Printf("Error getting current directory: %v\n", err)
			return
		}

		newNodes, err := scanDirectory(root)
		if err != nil {
			fmt.Printf("Error scanning directory: %v\n", err)
			return
		}

		syncedTree, err := syncFileTree(oldTree, newNodes)
		if err != nil {
			fmt.Printf("Error syncing filetree: %v\n", err)
			return
		}

		var targetNode FileNode
		for _, node := range syncedTree.Nodes {
			if node.Path == path {
				targetNode = node
				break
			}
		}

		if targetNode.Path == "" {
			fmt.Printf("Error: Path not found in filetree: %s\n", path)
			return
		}

		err = commitNode(targetNode)
		if err != nil {
			fmt.Printf("Error committing: %v\n", err)
			return
		}

		err = saveYAML(syncedTree, "filetree.yaml")
		if err != nil {
			fmt.Printf("Error saving updated filetree: %v\n", err)
			return
		}

		fmt.Printf("Successfully committed changes for %s\n", path)

	case "commit-all":
		oldTree, err := loadYAML("filetree.yaml")
		if err != nil {
			fmt.Printf("Error loading filetree: %v\n", err)
			return
		}

		root, err := os.Getwd()
		if err != nil {
			fmt.Printf("Error getting current directory: %v\n", err)
			return
		}

		newNodes, err := scanDirectory(root)
		if err != nil {
			fmt.Printf("Error scanning directory: %v\n", err)
			return
		}

		syncedTree, err := syncFileTree(oldTree, newNodes)
		if err != nil {
			fmt.Printf("Error syncing filetree: %v\n", err)
			return
		}

		err = commitAll(syncedTree.Nodes)
		if err != nil {
			fmt.Printf("Error in commit-all: %v\n", err)
			return
		}

		err = saveYAML(syncedTree, "filetree.yaml")
		if err != nil {
			fmt.Printf("Error saving updated filetree: %v\n", err)
			return
		}

		fmt.Println("Successfully committed all changes")

	default:
		fmt.Println("Invalid command")
	}
}

Temporary line added at 2024-07-17T17:05:12-06:00
