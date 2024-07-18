package core

func (ft *FileTree) AddNode(node *PathNode) {
	ft.Nodes[node.Path] = node
}

func (ft *FileTree) GetNode(path string) *PathNode {
	return ft.Nodes[path]
}

func ReadFileTreeFromYaml(filename string) (*FileTree, error)

func WriteFileTreeToYaml(ft *FileTree, filename string) error
