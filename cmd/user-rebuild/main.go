package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rizdev/devbase/internal/user"
)

func main() {
	// Default base directory is /devbase
	baseDir := "/devbase"
	if len(os.Args) > 1 {
		baseDir = os.Args[1]
	}

	// Create user manager
	manager := user.New(baseDir)

	// Ensure internal directory exists
	internalDir := filepath.Join(baseDir, ".internal")
	if err := os.MkdirAll(internalDir, 0700); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to create internal directory: %v\n", err)
		os.Exit(1)
	}

	// Rebuild users from persistent store
	fmt.Println("Rebuilding users from persistent store...")
	if err := manager.RebuildUsersFromStore(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to rebuild users: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("User rebuild completed successfully")

	// Check for orphaned home directories
	orphans, err := manager.FindOrphanedHomeDirs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to check for orphaned directories: %v\n", err)
	} else if len(orphans) > 0 {
		fmt.Println("\n=================================================")
		fmt.Println("WARNING: Found orphaned home directories:")
		fmt.Println("=================================================")
		for _, orphan := range orphans {
			fmt.Printf("  - /devbase/%s\n", orphan)
		}
		fmt.Println("\nThese directories exist but the corresponding")
		fmt.Println("system users were not found in the persistent store.")
		fmt.Println("\nTo recreate these users, use the web UI at")
		fmt.Println("http://localhost:8080 to recreate them with new")
		fmt.Println("passwords. The existing home directories will be")
		fmt.Println("preserved and reused.")
		fmt.Println("=================================================")
	}
}
