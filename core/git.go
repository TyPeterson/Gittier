package core

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const FileTreeBranch = "gittier"

// ---------- IsGitRepo ----------
func IsGitRepo() bool {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	err := cmd.Run()
	return err == nil
}

// ---------- BranchExists ----------
func BranchExists(branch string) bool {
	cmd := exec.Command("git", "branch", "--list", branch)
	output, err := cmd.Output()
	return err == nil && len(output) > 0
}

// ---------- GetCurrentBranch ----------
func GetCurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// ---------- CreateBranch ----------
func CreateBranch(branch string) error {
	cmd := exec.Command("git", "branch", branch, "main")
	return cmd.Run()
}

// ---------- DeleteBranch ----------
func DeleteBranch(branch string) error {
	cmd := exec.Command("git", "branch", "-D", branch)
	return cmd.Run()
}

// ---------- SwitchToBranch ----------
func SwitchToBranch(branch string) error {
	cmd := exec.Command("git", "switch", branch)
	return cmd.Run()
}

// ---------- Stash ----------
func Stash() error {
	cmd := exec.Command("git", "stash")
	return cmd.Run()
}

// ---------- StashPop ----------
func StashPop() error {
	cmd := exec.Command("git", "stash", "pop")
	return cmd.Run()
}

// ---------- NeedToStash ----------
func NeedToStash(branch string) (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to get git status: %w", err)
	}

	return len(strings.TrimSpace(string(output))) > 0, nil
}

// ---------- GetCommitHash ----------
func GetCommitHash(branch string) (string, error) {
	cmd := exec.Command("git", "rev-parse", branch)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current commit hash: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// ---------- GetFileTreeFromLsTree ----------
func GetFileTreeFromBranch(branch string) (*FileTree, error) {
	cmd := exec.Command("git", "ls-tree", "-r", "-t", "--name-only", branch)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get ls-tree output: %w", err)
	}

	commitHash, err := GetCommitHash(branch)
	if err != nil {
		return nil, err
	}

	fileTree := NewFileTree(commitHash)

	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	for scanner.Scan() {
		path := scanner.Text()

		// check if the path is a directory
		isDir := false
		fullPath := filepath.Join(".", path) // Prepend current directory
		fileInfo, err := os.Stat(fullPath)
		if err == nil {
			isDir = fileInfo.IsDir()
		} else if !os.IsNotExist(err) {
			// If there's an error other than "not exists", log it but continue
			fmt.Printf("Warning: Error checking %s: %v\n", fullPath, err)
		}

		// ensure the parent directories are added
		dirs := strings.Split(filepath.Dir(path), string(filepath.Separator))
		currentPath := ""
		for _, dir := range dirs {
			currentPath = filepath.Join(currentPath, dir)
			if currentPath != "." && !fileTree.HasNode(currentPath) {
				fileTree.AddNode(NewPathNode(currentPath, true))
			}
		}

		// add the file or directory
		fileTree.AddNode(NewPathNode(path, isDir))
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning ls-tree output: %w", err)
	}

	return fileTree, nil
}

// ---------- GetDiffOutput ----------
func GetDiffOutput(oldCommit string) ([]string, error) {
	cmd := exec.Command("git", "diff", "--name-status", oldCommit, "refs/heads/main")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return strings.Split(strings.TrimSpace(string(output)), "\n"), nil
}

// ---------- ProcessGitDiff ----------
func ProcessGitDiff(oldFileTree *FileTree, diffOutput []string) (*FileTree, error) {
	updatedFileTree := oldFileTree.Clone()

	for _, line := range diffOutput {
		parts := strings.Split(line, "\t")
		if len(parts) < 2 {
			continue
		}

		changeType := parts[0]
		switch changeType[0] {
		case 'A':
			newPath := parts[1]
			newNode := NewPathNode(newPath, false)
			updatedFileTree.AddNode(newNode)
		case 'D':
			oldPath := parts[1]
			updatedFileTree.DeleteNode(oldPath)
		case 'R':
			if len(parts) < 3 {
				continue
			}
			oldPath, newPath := parts[1], parts[2]
			updatedFileTree.UpdateNodePath(oldPath, newPath)
		}
	}

	return updatedFileTree, nil
}

// ---------- SyncFileTree ----------
func SyncFileTree(updatedFileTree, currentFileTree *FileTree) *FileTree {
	syncedFileTree := NewFileTree(currentFileTree.CommitHash)

	dfsOrder := GetDfsOrder(currentFileTree)
	for _, node := range dfsOrder {
		if updatedNode, exists := updatedFileTree.Nodes[node.Path]; exists {
			syncedFileTree.AddNode(updatedNode)
		} else {
			newNode := NewPathNode(node.Path, node.IsDir)
			syncedFileTree.AddNode(newNode)
		}
	}
	return syncedFileTree
}

// ---------- gitRename ----------
func gitRename(oldPath, newPath string) error {
	cmd := exec.Command("git", "mv", oldPath, newPath)
	return cmd.Run()
}

// ---------- CommitFolder ----------
func CommitFolderDescription(node *PathNode) error {
	tempFile := filepath.Join(node.Path, ".temp_commit_file")

	if err := os.WriteFile(tempFile, []byte("temporary content"), 0644); err != nil {
		return err
	}

	if err := StageAndCommit(".", fmt.Sprintf("%s temp commit", node.Path)); err != nil {
		return err
	}

	if err := os.Remove(tempFile); err != nil {
		return err
	}

	return StageAndCommit(".", node.Description)
}

// ---------- CommitFile ----------
func CommitFileDescription(node *PathNode) error {

	tempFileName, err := renameFile(node.Path)
	if err != nil {
		return err
	}

	if err := Commit(node.Description); err != nil {
		return err
	}

	if err := gitRename(tempFileName, node.Path); err != nil {
		return err
	}

	return nil
}

// ---------- Stage ----------
func Stage(path string) error {
	cmd := exec.Command("git", "add", path)
	return cmd.Run()
}

// ---------- Commit ----------
func Commit(message string) error {
	commitCmd := exec.Command("git", "commit", "-m", message)
	if err := commitCmd.Run(); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	return nil
}

// ---------- StageAndCommit ----------
func StageAndCommit(path, message string) error {
	if err := Stage(path); err != nil {
		fmt.Println("Error staging changes")
		return err
	}

	if err := Commit(message); err != nil {
		fmt.Println("Error committing changes")
		return err
	}

	return nil
}

// ---------- StageAndCommitBulk ----------
// func StageAndCommitBulk(path, message string) error

// ---------- mergeBranch ----------
func mergeBranch(targetBranch, sourceBranch string) error {
	originalBranch, err := GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	if originalBranch != targetBranch {
		if err := Stash(); err != nil {
			return fmt.Errorf("failed to stash changes: %w", err)
		}
		if err := SwitchToBranch(targetBranch); err != nil {
			return fmt.Errorf("failed to switch to branch %s: %w", targetBranch, err)
		}

		defer func() {
			if err := SwitchToBranch(originalBranch); err != nil {
				fmt.Printf("Warning: failed to switch back to branch %s: %v\n", originalBranch, err)
			}

			if err := StashPop(); err != nil {
				fmt.Printf("Warning: failed to pop stash: %v\n", err)
			}
		}()
	}

	cmd := exec.Command("git", "merge", sourceBranch)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to merge branch %s into branch %s: %w\n%s", sourceBranch, targetBranch, err, string(output))
	}

	fmt.Printf("Merge successful. Output: %s\n", string(output))
	return nil
}
