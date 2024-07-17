package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"sort"
	"strings"
	"time"

	gitignore "github.com/sabhiram/go-gitignore"
	// "golang.org/x/sys/unix"
	"gopkg.in/yaml.v2"
)

type FileNode struct {
	Path        string `yaml:"path"`
	Description string `yaml:"description"`
	IsDir       bool   `yaml:"is_dir"`
}

func scanDirectory(root string) ([]FileNode, error) {
	var nodes []FileNode
	visited := make(map[string]bool)

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
		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		// Skip if already visited
		if visited[relPath] {
			return nil
		}

		entries, err := os.ReadDir(path)
		if err != nil {
			return err
		}

		// Sort entries to ensure consistent ordering
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Name() < entries[j].Name()
		})

		var dirChildren []FileNode

		for _, entry := range entries {
			fullPath := filepath.Join(path, entry.Name())
			childRelPath, err := filepath.Rel(root, fullPath)
			if err != nil {
				return err
			}

			// Skip .git directory
			if childRelPath == ".git" || strings.HasPrefix(childRelPath, ".git"+string(os.PathSeparator)) {
				continue
			}

			// Check if file is ignored by .gitignore
			if ignore != nil && ignore.MatchesPath(childRelPath) {
				continue
			}

			info, err := entry.Info()
			if err != nil {
				return err
			}

			if info.IsDir() {
				if err := dfs(fullPath); err != nil {
					return err
				}
			} else {
				fileNode := FileNode{
					Path:        childRelPath,
					Description: "No description added",
					IsDir:       false,
				}
				dirChildren = append(dirChildren, fileNode)
			}
		}

		// Add children first
		nodes = append(nodes, dirChildren...)

		// Add the directory itself if it's not the root
		if relPath != "." {
			dirNode := FileNode{
				Path:        relPath,
				Description: "No description added",
				IsDir:       true,
			}
			nodes = append(nodes, dirNode)
		}

		visited[relPath] = true
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

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return err
	}

	gitignorePath := ".gitignore"
	var lines []string

	if file, err := os.Open(gitignorePath); err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
	}

	found := false
	for _, line := range lines {
		if strings.TrimSpace(line) == filename {
			found = true
			break
		}
	}

	if !found {
		lines = append(lines, filename)
		content := strings.Join(lines, "\n")
		if err := os.WriteFile(gitignorePath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to update .gitignore: %w", err)
		}
	}

	return nil
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

func getGitRenames() (map[string]string, error) {
	cmd := exec.Command("git", "diff", "--name-status", "--cached")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("error running git diff: %w", err)
	}

	renames := make(map[string]string)
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		parts := strings.Split(line, "\t")
		if len(parts) == 3 && strings.HasPrefix(parts[0], "R") {
			oldPath, newPath := parts[1], parts[2]
			renames[newPath] = oldPath
		}
	}

	// print out the renames that were detected (if any)
	if len(renames) > 0 {
		fmt.Println("Detected renames:")
		for newPath, oldPath := range renames {
			fmt.Printf("  %s -> %s\n", oldPath, newPath)
		}
	} else {
		fmt.Println("No renames detected")
	}

	return renames, nil
}

func syncFileTree(oldNodes []FileNode, newNodes []FileNode) ([]FileNode, error) {
	oldMap := make(map[string]FileNode)
	for _, node := range oldNodes {
		oldMap[node.Path] = node
	}

	renames, err := getGitRenames()
	if err != nil {
		return nil, err
	}

	var syncedNodes []FileNode
	for _, newNode := range newNodes {
		if oldPath, isRenamed := renames[newNode.Path]; isRenamed {
			if oldNode, exists := oldMap[oldPath]; exists {
				newNode.Description = oldNode.Description
			}
		} else if oldNode, exists := oldMap[newNode.Path]; exists {
			newNode.Description = oldNode.Description
		}
		syncedNodes = append(syncedNodes, newNode)
	}

	return syncedNodes, nil
}

func commitFile(filePath string, description string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	tempLine := fmt.Sprintf("\nTemporary line added at %s\n", time.Now().Format(time.RFC3339))
	newContent := append(content, []byte(tempLine)...)

	if err := os.WriteFile(filePath, newContent, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	cmd := exec.Command("git", "add", filePath)
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
	tempFileName := fmt.Sprintf("temp_file_%d.txt", time.Now().UnixNano())
	tempFilePath := filepath.Join(dirPath, tempFileName)
	tempFile, err := os.Create(tempFilePath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tempFile.WriteString("Temporary file for commit")
	tempFile.Close()

	cmd := exec.Command("git", "add", tempFilePath)
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

		err = saveYAML(nodes, "filetree.yaml")
		if err != nil {
			fmt.Printf("Error saving filetree: %v\n", err)
			return
		}

		fmt.Println("Filetree initialized")

	case "commit", "commit-all":
		oldNodes, err := loadYAML("filetree.yaml")
		if err != nil {
			fmt.Printf("Error loading filetree: %v\n", err)
			return
		}

		root, err := os.Getwd()
		if err != nil {
			fmt.Printf("Error getting current directory: %v\n", err)
			return
		}

		// Stage all changes
		stageCmd := exec.Command("git", "add", "-A")
		if err := stageCmd.Run(); err != nil {
			fmt.Printf("Error staging changes: %v\n", err)
			return
		}

		newNodes, err := scanDirectory(root)
		if err != nil {
			fmt.Printf("Error scanning directory: %v\n", err)
			return
		}

		syncedNodes, err := syncFileTree(oldNodes, newNodes)
		if err != nil {
			fmt.Printf("Error syncing filetree: %v\n", err)
			return
		}

		err = saveYAML(syncedNodes, "filetree.yaml")
		if err != nil {
			fmt.Printf("Error saving updated filetree: %v\n", err)
			return
		}

		if command == "commit" {
			if len(os.Args) < 3 {
				fmt.Println("Usage: go run main.go commit <path>")
				return
			}
			path := os.Args[2]

			var targetNode FileNode
			for _, node := range syncedNodes {
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

			fmt.Printf("Successfully committed changes for %s\n", path)
		} else {
			err = commitAll(syncedNodes)
			if err != nil {
				fmt.Printf("Error in commit-all: %v\n", err)
				return
			}

			fmt.Println("Successfully committed all changes")
		}

	default:
		fmt.Println("Invalid command")
	}
}

Temporary line added at 2024-07-17T15:30:53-06:00
