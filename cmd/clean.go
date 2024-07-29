package cmd

import (
	"github.com/TyPeterson/Gittier/core"
)

func Clean() error {
	// delete the FileTreeBranch branch
	if err := core.DeleteBranch(core.FileTreeBranch); err != nil {
		return err
	}

	return nil
}
