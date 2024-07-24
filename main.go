package main

import (
	"fmt"
	"os"

	"github.com/TyPeterson/Gittier/cmd"
	"github.com/TyPeterson/Gittier/core"
)

func main() {
	if len(os.Args) < 2 {
		core.PrintUsage()
		os.Exit(1)
	}

	var err error = nil
	switch os.Args[1] {
	case "init":
		err = cmd.Init()
	case "sync":
		err = cmd.Sync()
	case "desc":
		if len(os.Args) < 4 {
			fmt.Println("Usage: filetree desc <path> <description>")
			os.Exit(1)
		}
		err = cmd.Desc(os.Args[2], os.Args[3], true)
	case "commit":
		err = cmd.Commit()
	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		core.PrintUsage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

}
