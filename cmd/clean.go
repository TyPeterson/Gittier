package cmd

import (
	"github.com/TyPeterson/Gittier/core"
)

func Clean() error {
	// delete the FileTreeBranch branch
	if err := core.DeleteBranch(core.FileTreeBranch); err != nil {
		return err
	}

	// delete the filetree.yaml file
	if err := core.DeleteFile("filetree.yaml"); err != nil {
		return err
	}

	// delete the .gitattributes file
	if err := core.DeleteFile(".gitattributes"); err != nil {
		return err
	}

	return nil
}
