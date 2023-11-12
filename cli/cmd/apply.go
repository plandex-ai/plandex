package cmd

import (
	"fmt"
	"os"
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
		return fmt.Errorf("error: %v", err)
	}

	if name == "" || name == "current" {
		name = lib.CurrentPlanName
	}

	rootDir := filepath.Join(plandexDir, name)

	if _, err := os.Stat(rootDir); os.IsNotExist(err) {
		return fmt.Errorf("error: plan with name '%+v' does not exist", name)
	}

	// isRepo := lib.CwdIsGitRepo()

	// if isRepo {

	// 	planState, err := lib.GetPlanState()
	// 	if err != nil {
	// 		return fmt.Errorf("error loading plan state: %v", err)
	// 	}

	// }

	copiedAny := false
	// Enumerate all paths in [draftDir]/files
	filesDir := filepath.Join(rootDir, "draft", "files")
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
		dstPath := filepath.Join(lib.ProjectRoot, relPath)

		// Copy the file
		err = lib.CopyFile(srcPath, dstPath)
		if err != nil {
			return fmt.Errorf("failed to copy %s to %s: %w", srcPath, dstPath, err)
		}
		copiedAny = true
		return nil
	})

	if err != nil {
		return fmt.Errorf("error processing files: %v", err)
	}

	if copiedAny {
		fmt.Println("✅ Applied changes")
	} else {
		return fmt.Errorf("this plan has no changes to apply")
	}

	return nil
}
