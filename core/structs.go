package core

type FileTree struct {
	CommitHash string               `yaml:"commit_hash"`
	Nodes      map[string]*PathNode `yaml:"nodes"`
}

type PathNode struct {
	Path        string      `yaml:"path"`
	Description string      `yaml:"description"`
	IsDir       bool        `yaml:"is_dir"`
	Children    []*PathNode `yaml:"children,omitempty"`
}

// ---------- NewFileTree ----------
func NewFileTree(commitHash string) *FileTree {
	return &FileTree{
		CommitHash: commitHash,
		Nodes:      make(map[string]*PathNode),
	}
}

// ---------- NewPathNode ----------
func NewPathNode(lsTreeItem string, isDir bool) *PathNode {
	return &PathNode{
		Path:        lsTreeItem,
		Description: "no description added",
		IsDir:       isDir,
	}
}

// Temporary line for commit
