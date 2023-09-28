package cmd

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"plandex/lib"

	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(applyCmd)
}

var applyCmd = &cobra.Command{
	Use:   "apply [name]",
	Short: "Apply a plan to the project",
	Args:  cobra.ExactArgs(1),
	Run:   apply,
}

func apply(cmd *cobra.Command, args []string) {
	plandexDir, _, err := lib.FindOrCreatePlandex()
	name := args[0]

	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		return
	}

	if name == "" {
		name = lib.CurrentPlanName
	}

	if name == "" {
		log.Fatalln("No plan name provided and no current plan is set.")
	}

	// Check git installed
	if !lib.IsCommandAvailable("git") {
		log.Fatalln("Error: git is required")
	}

	rootDir := filepath.Join(plandexDir, name)

	if _, err := os.Stat(rootDir); os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, "Error: plan with name '"+name+"' does not exist")
		return
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
		fmt.Fprintln(os.Stderr, "Error processing files:", err)
		return
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
			fmt.Fprintln(os.Stderr, "Error executing plan:", err)
			return
		}
		didExec = true
	}

	if copiedAny || didExec {
		fmt.Println("Plan applied successfully!")
	} else {
		fmt.Fprintf(os.Stderr, "This plan has no changes to apply.")
		os.Exit(1)
	}

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
