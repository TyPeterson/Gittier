package core

import (
	"fmt"
	"strings"
)

func FileExists(filename string) bool

func ConfirmOverwrite(filename string) bool

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
