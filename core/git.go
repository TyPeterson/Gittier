package core

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	gitignore "github.com/sabhiram/go-gitignore"
)

// ---------- IsGitRepo ----------
func IsGitRepo() bool {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	err := cmd.Run()
	return err == nil
}

// ---------- AddToGitignore ----------
func AddToGitignore(pattern string) error {
	gitignorePath := ".gitignore"
	var lines []string

	if _, err := os.Stat(gitignorePath); err == nil {
		file, err := os.Open(gitignorePath)
		if err != nil {
			return fmt.Errorf("failed to open .gitignore: %w", err)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}

		if err := scanner.Err(); err != nil {
			return fmt.Errorf("failed to read .gitignore: %w", err)
		}
	}

	ignoreFile := gitignore.CompileIgnoreLines(lines...)
	if ignoreFile.MatchesPath(pattern) {
		// Pattern or a superset of it already exists, no need to add
		return nil
	}

	lines = append(lines, pattern)

	content := strings.Join(lines, "\n")
	if err := os.WriteFile(gitignorePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write to .gitignore: %w", err)
	}

	return nil
}

// ---------- GetCurrentCommitHash ----------
func GetCurrentCommitHash() (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current commit hash: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// ---------- GetFileTreeFromLsTree ----------
func GetFileTreeFromLsTree() (*FileTree, error) {
	cmd := exec.Command("git", "ls-tree", "-r", "-t", "--name-only", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get ls-tree output: %w", err)
	}

	commitHash, err := GetCurrentCommitHash()
	if err != nil {
		return nil, err
	}

	fileTree := NewFileTree(commitHash)
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	for scanner.Scan() {
		path := scanner.Text()

		// Check if the path is a directory
		isDir := false
		fullPath := filepath.Join(".", path) // Prepend current directory
		fileInfo, err := os.Stat(fullPath)
		if err == nil {
			isDir = fileInfo.IsDir()
		} else if !os.IsNotExist(err) {
			// If there's an error other than "not exists", log it but continue
			fmt.Printf("Warning: Error checking %s: %v\n", fullPath, err)
		}

		// Ensure the parent directories are added
		dirs := strings.Split(filepath.Dir(path), string(filepath.Separator))
		currentPath := ""
		for _, dir := range dirs {
			currentPath = filepath.Join(currentPath, dir)
			if currentPath != "." && !fileTree.HasNode(currentPath) {
				fileTree.AddNode(NewPathNode(currentPath, true))
			}
		}

		// Add the file or directory
		fileTree.AddNode(NewPathNode(path, isDir))
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning ls-tree output: %w", err)
	}

	return fileTree, nil
}

// ---------- HasNode ----------
func (ft *FileTree) HasNode(path string) bool {
	_, exists := ft.Nodes[path]
	return exists
}

// ---------- GetDiffOutput ----------
func GetDiffOutput(oldCommit string) ([]string, error) {
	cmd := exec.Command("git", "diff", "--name-status", oldCommit, "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return strings.Split(strings.TrimSpace(string(output)), "\n"), nil
}

// ---------- ProcessGitDiff ----------
func ProcessGitDiff(oldFileTree *FileTree, diffOutput []string) (*FileTree, error) {
	updatedFileTree := oldFileTree.Clone()

	for _, line := range diffOutput {
		parts := strings.Split(line, "\t")
		if len(parts) < 2 {
			continue
		}

		changeType := parts[0]
		switch changeType[0] {
		case 'A':
			newPath := parts[1]
			newNode := NewPathNode(newPath, false)
			updatedFileTree.AddNode(newNode)
		case 'D':
			oldPath := parts[1]
			updatedFileTree.DeleteNode(oldPath)
		case 'R':
			if len(parts) < 3 {
				continue
			}
			oldPath, newPath := parts[1], parts[2]
			updatedFileTree.UpdateNodePath(oldPath, newPath)
		}
	}

	return updatedFileTree, nil
}

// ---------- SyncFileTree ----------
func SyncFileTree(updatedFileTree, currentFileTree *FileTree) *FileTree {
	syncedFileTree := NewFileTree(currentFileTree.CommitHash)

	dfsOrder := GetDfsOrder(currentFileTree)
	for _, node := range dfsOrder {
		if updatedNode, exists := updatedFileTree.Nodes[node.Path]; exists {
			syncedFileTree.AddNode(updatedNode)
		} else {
			newNode := NewPathNode(node.Path, node.IsDir)
			syncedFileTree.AddNode(newNode)
		}
	}
	return syncedFileTree
}
