package core

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ---------- FileExists ----------
func FileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// ---------- GetChildNodes ----------
func (ft *FileTree) GetChildNodes(path string) []*PathNode {
	var children []*PathNode
	for nodePath, node := range ft.Nodes {
		parentPath := getParentPath(nodePath)
		if parentPath == path {
			children = append(children, node)
		}
	}
	return children
}

// ---------- IsAncestor ----------
func (ft *FileTree) IsAncestor(potentialAncestor, path string) bool {
	return strings.HasPrefix(path, potentialAncestor+"/")
}

// ---------- GetParentPath ----------
func getParentPath(path string) string {
	return path[:strings.LastIndex(path, "/")]
}

// ---------- GetDfsOrder ----------
func GetDfsOrder(ft *FileTree) []*PathNode {
	var result []*PathNode
	visited := make(map[string]bool)

	var dfs func(path string)
	dfs = func(path string) {
		if visited[path] {
			return
		}
		visited[path] = true

		node, exists := ft.Nodes[path]
		if !exists {
			return
		}

		// For directories, visit children first
		if node.IsDir {
			children := getChildrenPaths(ft, path)
			sort.Strings(children) // Sort children for consistent ordering
			for _, childPath := range children {
				dfs(childPath)
			}
		}

		// Add the node to result after visiting children
		result = append(result, node)
	}

	// Get and sort top-level paths
	topLevelPaths := getTopLevelPaths(ft)
	sort.Strings(topLevelPaths)

	// Start DFS from each top-level path
	for _, path := range topLevelPaths {
		dfs(path)
	}

	return result
}

// ---------- getChildrenPaths ----------
func getChildrenPaths(ft *FileTree, parentPath string) []string {
	var children []string
	for path := range ft.Nodes {
		if filepath.Dir(path) == parentPath && path != parentPath {
			children = append(children, path)
		}
	}
	return children
}

// ---------- getTopLevelPaths ----------
func getTopLevelPaths(ft *FileTree) []string {
	var topLevelPaths []string
	for path := range ft.Nodes {
		if !strings.Contains(path, string(filepath.Separator)) {
			topLevelPaths = append(topLevelPaths, path)
		}
	}
	return topLevelPaths
}

// ---------- Clone ----------
func (ft *FileTree) Clone() *FileTree {
	newTree := NewFileTree(ft.CommitHash)
	for path, node := range ft.Nodes {
		newNode := &PathNode{
			Path:        node.Path,
			Description: node.Description,
			IsDir:       node.IsDir,
		}
		newTree.Nodes[path] = newNode
	}
	return newTree
}

// ---------- PrintUsage ----------
func PrintUsage() {
	fmt.Println("Usage: filetree <command> [arguments]")
	fmt.Println("\nAvailable commands:")
	fmt.Println("  init                  Initialize a new filetree.yaml")
	fmt.Println("  update                Update the existing filetree.yaml")
	fmt.Println("  desc <path> <description>  Add or update description for a path")
}

// ---------- appendTempLine ----------
func appendTempLine(filePath string) error {
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString("\n// Temporary line for commit\n")
	return err
}

// ---------- removeTempLine ----------
func removeTempLine(filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	if len(lines) < 2 {
		return nil
	}

	newContent := strings.Join(lines[:len(lines)-2], "\n")
	return os.WriteFile(filePath, []byte(newContent), 0644)
}

// ---------- renameFile ----------
func renameFile(filePath string) (string, error) {
	// add _temp to the file name (e.g. filetree.go -> filetree_temp.go)
	dir := filepath.Dir(filePath)
	filename := filepath.Base(filePath)
	ext := filepath.Ext(filename)
	nameWithoutExt := strings.TrimSuffix(filename, ext)

	newFilename := nameWithoutExt + "_temp" + ext
	newFilePath := filepath.Join(dir, newFilename)

	err := gitRename(filePath, newFilePath)
	if err != nil {
		return "", err
	}

	return newFilePath, nil
}

// ---------- DeleteFile ----------
func DeleteFile(filePath string) error {
	err := os.Remove(filePath)
	if err != nil {
		return err
	}
	return nil
}
