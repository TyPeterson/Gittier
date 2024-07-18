package main

import (
	"fmt"
	"os"

	"github.com/TyPeterson/Gittier/cmd"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	var err error = nil
	switch os.Args[1] {
	case "init":
		err = cmd.Init()
	// case "update":
	//     err := cmd.Update()
	// case "desc":
	//     if len(os.Args) < 4 {
	//         fmt.Println("Usage: filetree desc <path> <description>")
	//         os.Exit(1)
	//     }
	//     err := cmd.Desc(os.Args[2], os.Args[3])
	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: filetree <command> [arguments]")
	fmt.Println("\nAvailable commands:")
	fmt.Println("  init                  Initialize a new filetree.yaml")
	fmt.Println("  update                Update the existing filetree.yaml")
	fmt.Println("  desc <path> <description>  Add or update description for a path")
}
