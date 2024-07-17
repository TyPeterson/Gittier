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
	"gopkg.in/yaml.v2"
)

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

	// Write YAML data to file
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return err
	}

	// Add filename to .gitignore if it's not already there
	gitignorePath := ".gitignore"
	var lines []string

	// Read existing .gitignore file
	if file, err := os.Open(gitignorePath); err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
	}

	// Check if filename is already in .gitignore
	found := false
	for _, line := range lines {
		if strings.TrimSpace(line) == filename {
			found = true
			break
		}
	}

	// If filename is not in .gitignore, append it
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

func printDFS(nodes []FileNode) {

	for _, node := range nodes {
		fmt.Println(node.Path, ":", node.Description)
	}
}

func commitFile(filePath string, description string) error {
	// Read the entire file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Append a temporary line to the file
	tempLine := fmt.Sprintf("\nTemporary line added at %s\n", time.Now().Format(time.RFC3339))
	newContent := append(content, []byte(tempLine)...)

	// Write the file with the appended line
	if err := os.WriteFile(filePath, newContent, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	// Git add the file
	cmd := exec.Command("git", "add", filePath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git add failed: %w", err)
	}

	// Git commit the temporary change
	cmd = exec.Command("git", "commit", "-m", fmt.Sprintf("%s temp change", filepath.Base(filePath)))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git commit (temp) failed: %w", err)
	}

	// Remove the temporary line by writing the original content back
	if err = os.WriteFile(filePath, content, 0644); err != nil {
		return fmt.Errorf("failed to restore original content: %w", err)
	}

	// Git add the file again
	cmd = exec.Command("git", "add", filePath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git add (final) failed: %w", err)
	}

	// Git commit with the description
	cmd = exec.Command("git", "commit", "-m", description)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git commit (final) failed: %w", err)
	}

	return nil
}

func commitDirectory(dirPath string, description string) error {
	// Create a temporary file in the directory
	tempFileName := fmt.Sprintf("temp_file_%d.txt", time.Now().UnixNano())
	tempFilePath := filepath.Join(dirPath, tempFileName)
	tempFile, err := os.Create(tempFilePath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tempFile.WriteString("Temporary file for commit")
	tempFile.Close()

	// Git add the temporary file
	cmd := exec.Command("git", "add", tempFilePath)
	if err := cmd.Run(); err != nil {
		os.Remove(tempFilePath) // Clean up the temp file
		return fmt.Errorf("git add (temp file) failed: %w", err)
	}

	// Git commit the temporary file
	cmd = exec.Command("git", "commit", "-m", fmt.Sprintf("Add temporary file in %s", dirPath))
	if err := cmd.Run(); err != nil {
		os.Remove(tempFilePath) // Clean up the temp file
		return fmt.Errorf("git commit (temp file) failed: %w", err)
	}

	// Remove the temporary file
	if err := os.Remove(tempFilePath); err != nil {
		return fmt.Errorf("failed to remove temp file: %w", err)
	}

	// Git add the removal of the temporary file
	cmd = exec.Command("git", "add", tempFilePath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git add (remove temp file) failed: %w", err)
	}

	// Git commit with the directory description
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

	case "dfs":

	case "commit":
		if len(os.Args) < 3 {
			fmt.Println("Usage: go run main.go commit <path>")
			return
		}
		path := os.Args[2]

		nodes, err := loadYAML("filetree.yaml")
		if err != nil {
			fmt.Printf("Error loading filetree: %v\n", err)
			return
		}

		var targetNode FileNode
		for _, node := range nodes {
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

	case "commit-all":
		nodes, err := loadYAML("filetree.yaml")
		if err != nil {
			fmt.Printf("Error loading filetree: %v\n", err)
			return
		}

		err = commitAll(nodes)
		if err != nil {
			fmt.Printf("Error in commit-all: %v\n", err)
			return
		}

		fmt.Println("Successfully committed all changes")

	default:
		fmt.Println("Invalid command")
	}
}
