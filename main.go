package main

import (
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

func commitFile(filePath string, nodes []FileNode) error {
	// Find the file in our nodes
	var fileNode FileNode
	for _, node := range nodes {
		if node.Path == filePath {
			fileNode = node
			break
		}
	}
	if fileNode.Path == "" {
		return fmt.Errorf("file not found in YAML: %s", filePath)
	}

	// Add a temporary line to the file
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	tempLine := fmt.Sprintf("Temporary line added at %s\n", time.Now().Format(time.RFC3339))
	if _, err := f.WriteString(tempLine); err != nil {
		f.Close()
		return err
	}
	f.Close()

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

	// Remove the temporary line
	input, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	lines := strings.Split(string(input), "\n")
	output := strings.Join(lines[:len(lines)-2], "\n") // Remove last line and the empty line after it
	if err = os.WriteFile(filePath, []byte(output), 0644); err != nil {
		return err
	}

	// Git add the file again
	cmd = exec.Command("git", "add", filePath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git add (final) failed: %w", err)
	}

	// Git commit with the description
	cmd = exec.Command("git", "commit", "-m", fileNode.Description)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git commit (final) failed: %w", err)
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
				fmt.Println("Usage: go run main.go commit <filepath>")
				return
			}
			filePath := os.Args[2]

			nodes, err := loadYAML("filetree.yaml")
			if err != nil {
				fmt.Printf("Error loading filetree: %v\n", err)
				return
			}

			err = commitFile(filePath, nodes)
			if err != nil {
				fmt.Printf("Error committing file: %v\n", err)
				return
			}

			fmt.Printf("Successfully committed changes for %s\n", filePath)

	default:
		fmt.Println("Invalid command")
	}
}
