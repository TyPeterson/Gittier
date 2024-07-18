package core

import (
	"bufio"
	"fmt"
	"os"
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

// ---------- ConfirmOverwrite ----------
func ConfirmOverwrite(filename string) bool {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("File '%s' already exists. Overwrite? (y/n): ", filename)
		response, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading input: ", err)
			return false
		}

		response = strings.ToLower(strings.TrimSpace(response))
		switch response {
		case "y", "yes":
			return true
		case "n", "no":
			return false
		default:
			fmt.Println("Invalid response. Please enter 'y' or 'n'.")
		}
	}
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

// ---------- PrintTree ----------
func (ft *FileTree) PrintTree() {
	for path, node := range ft.Nodes {
		indent := strings.Repeat(" ", strings.Count(path, "/")*2)
		fmt.Printf("%s%s (%s)\n", indent, node.Path, node.Description)
	}
}
