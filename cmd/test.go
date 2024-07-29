package cmd

import (
	"fmt"

	"github.com/TyPeterson/Gittier/core"
)

func Test() error {
	fmt.Println("Testing: NeedToStash")
	currentBranch, err := core.GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	needToStash, err := core.NeedToStash(currentBranch)
	if err != nil {
		return fmt.Errorf("failed to check if need to stash: %w", err)
	}

	fmt.Println("Need to stash:", needToStash)

	return nil
}
