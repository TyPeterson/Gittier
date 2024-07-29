package core

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v2"
)

// ---------- AddNode ----------
func (ft *FileTree) AddNode(node *PathNode) {
	ft.Nodes[node.Path] = node
}

// ---------- GetNode ----------
func (ft *FileTree) GetNode(path string) *PathNode {
	return ft.Nodes[path]
}

// ---------- DeleteNode ----------
func (ft *FileTree) DeleteNode(path string) error {
	if _, exists := ft.Nodes[path]; !exists {
		return fmt.Errorf("node does not exist: %s", path)
	}

	for nodePath := range ft.Nodes {
		if strings.HasPrefix(nodePath, path) {
			delete(ft.Nodes, nodePath)
		}
	}

	return nil
}

// ---------- UpdateNodePath ----------
func (ft *FileTree) UpdateNodePath(oldPath, newPath string) error {
	node, exists := ft.Nodes[oldPath]
	if !exists {
		return fmt.Errorf("node does not exist: %s", oldPath)
	}

	// update nodes as well as any children
	nodesToUpdate := make(map[string]*PathNode)
	for path, n := range ft.Nodes {
		if strings.HasPrefix(path, oldPath) {
			updatedPath := strings.Replace(path, oldPath, newPath, 1)
			n.Path = updatedPath
			nodesToUpdate[updatedPath] = n
			delete(ft.Nodes, path)
		}
	}

	// update the node within FileTree
	for path, n := range nodesToUpdate {
		ft.Nodes[path] = n
	}

	node.Path = newPath

	return nil
}

// ---------- UpdateNodeDescription ----------
func (ft *FileTree) UpdateNodeDescription(path, description string) error {
	node, exists := ft.Nodes[path]
	if !exists {
		return fmt.Errorf("node does not exist: %s", path)
	}

	node.Description = description
	return nil
}

// ---------- HasNode ----------
func (ft *FileTree) HasNode(path string) bool {
	_, exists := ft.Nodes[path]
	return exists
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

// ---------- ReadFileTreeFromYaml ----------
func ReadFileTreeFromYaml(filename string) (*FileTree, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error reading YAML file: %w", err)
	}

	// unmarshal the yaml directly into a FileTree
	var fileTree FileTree
	err = yaml.Unmarshal(data, &fileTree)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling YAML: %w", err)
	}

	return &fileTree, nil
}

// ---------- WriteFileTreeToYaml ----------
func WriteFileTreeToYaml(ft *FileTree, filename string) error {
	data, err := yaml.Marshal(ft)
	if err != nil {
		return fmt.Errorf("error marshaling YAML: %w", err)
	}

	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return fmt.Errorf("error writing YAML file: %w", err)
	}

	return nil
}
