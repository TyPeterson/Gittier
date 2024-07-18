package core

func IsGitRepo() bool

func AddToGitignore(filename string) error

func GetCurrentCommitHash() (string, error)

func GetLsTree() ([]string, error)

func GetFileTreeFromLsTree() (*FileTree, error)
