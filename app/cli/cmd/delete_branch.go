package cmd

import (
	"fmt"
	"plandex/api"
	"plandex/auth"
	"plandex/lib"
	"plandex/term"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var deleteBranchCmd = &cobra.Command{
	Use:     "delete-branch",
	Aliases: []string{"db"},
	Short:   "Delete a plan branch by name or index",
	Run:     deleteBranch,
	Args:    cobra.MaximumNArgs(1),
}

func init() {
	RootCmd.AddCommand(deleteBranchCmd)
}

func deleteBranch(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()
	lib.MustResolveProject()

	if lib.CurrentPlanId == "" {
		fmt.Println("🤷‍♂️ No current plan")
		return
	}

	var branch string
	var nameOrIdx string

	if len(args) > 0 {
		nameOrIdx = strings.TrimSpace(args[0])
	}

	if nameOrIdx == "main" {
		fmt.Println("🚨 Cannot delete main branch")
		return
	}

	term.StartSpinner("")
	branches, apiErr := api.Client.ListBranches(lib.CurrentPlanId)
	term.StopSpinner()

	if apiErr != nil {
		term.OutputErrorAndExit("Error getting branches: %v", apiErr)
		return
	}

	if nameOrIdx == "" {
		opts := make([]string, len(branches))
		for i, branch := range branches {
			if branch.Name == "main" {
				continue
			}
			opts[i] = branch.Name
		}

		if len(opts) == 0 {
			fmt.Println("🤷‍♂️ No branches to delete")
			return
		}

		sel, err := term.SelectFromList("Select a branch to delete", opts)

		if err != nil {
			term.OutputErrorAndExit("Error selecting branch: %v", err)
			return
		}

		branch = sel
	}

	// see if it's an index
	idx, err := strconv.Atoi(nameOrIdx)

	if err == nil {
		if idx > 0 && idx <= len(branches) {
			branch = branches[idx-1].Name
		} else {
			term.OutputErrorAndExit("Branch index out of range")
		}
	} else {
		for _, b := range branches {
			if b.Name == nameOrIdx {
				branch = b.Name
				break
			}
		}

		if branch == "" {
			term.OutputErrorAndExit("Branch not found")
		}
	}

	found := false
	for _, b := range branches {
		if b.Name == branch {
			found = true
			break
		}
	}

	if !found {
		fmt.Printf("🤷‍♂️ Branch %s does not exist\n", color.New(color.Bold, term.ColorHiCyan).Sprint(branch))
		return
	}

	term.StartSpinner("")
	apiErr = api.Client.DeleteBranch(lib.CurrentPlanId, branch)
	term.StopSpinner()

	if apiErr != nil {
		term.OutputErrorAndExit("Error deleting branch: %v", apiErr)
		return
	}

	fmt.Printf("✅ Deleted branch %s\n", color.New(color.Bold, term.ColorHiCyan).Sprint(branch))

	fmt.Println()
	term.PrintCmds("", "branches")
}
