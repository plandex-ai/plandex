package cmd

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"plandex/lib"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(applyCmd)
}

var applyCmd = &cobra.Command{
	Use:     "apply [name]",
	Aliases: []string{"ap"},
	Short:   "Apply a plan to the project",
	Args:    cobra.MaximumNArgs(1),
	RunE:    apply,
}

func apply(cmd *cobra.Command, args []string) error {
	plandexDir, _, err := lib.FindOrCreatePlandex()
	var name string

	if len(args) > 0 {
		name = args[0]
		name = strings.TrimSpace(name)
	}

	if err != nil {
		return fmt.Errorf("Error: %v", err)
	}

	if name == "" || name == "current" {
		name = lib.CurrentPlanName
	}

	// Check git installed
	if !lib.IsCommandAvailable("git") {
		log.Fatalln("Error: git is required")
	}

	rootDir := filepath.Join(plandexDir, name)

	if _, err := os.Stat(rootDir); os.IsNotExist(err) {
		return fmt.Errorf("Error: plan with name '%+v' does not exist", name)
	}

	copiedAny := false
	// Enumerate all paths in [planDir]/files
	filesDir := filepath.Join(rootDir, "plan", "files")
	err = filepath.Walk(filesDir, func(srcPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		// Compute relative path
		relPath, err := filepath.Rel(filesDir, srcPath)
		if err != nil {
			return err
		}

		// Compute destination path
		dstPath := filepath.Join(".", relPath)

		// Copy the file
		err = copyFile(srcPath, dstPath)
		if err != nil {
			return fmt.Errorf("failed to copy %s to %s: %w", srcPath, dstPath, err)
		}
		copiedAny = true
		return nil
	})

	if err != nil {
		return fmt.Errorf("Error processing files: %v", err)
	}

	didExec := false
	// If it exists, run [planDir]/exec.sh with cwd set to current process cwd
	execPath := filepath.Join(rootDir, "exec.sh")
	if _, err := os.Stat(execPath); err == nil {
		// exec.sh exists, run it
		cmd := exec.Command(execPath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Dir = "." // Current process CWD

		err = cmd.Run()
		if err != nil {
			return fmt.Errorf("Error executing plan: %v", err)
		}
		didExec = true
	}

	if copiedAny || didExec {
		fmt.Println("Plan applied successfully!")
	} else {
		return fmt.Errorf("This plan has no changes to apply.")
	}

	return nil
}

// Utility function to copy files
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}
