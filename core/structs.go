package core

type FileTree struct {
	CommitHash string
	Nodes      map[string]*PathNode
}

type PathNode struct {
	ID          int
	Path        string
	Description string
	IsDir       bool
}

func NewFileTree(commitHash string) *FileTree {
	return &FileTree{
		CommitHash: commitHash,
		Nodes:      make(map[string]*PathNode),
	}
}

func NewPathNode(lsTreeItem string, isDir bool) *PathNode {
	return &PathNode{
		Path:  lsTreeItem,
		IsDir: isDir,
	}
}
