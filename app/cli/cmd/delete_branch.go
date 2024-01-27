package cmd

import (
	"fmt"
	"plandex/api"
	"plandex/auth"
	"plandex/lib"
	"plandex/term"
	"strings"

	"github.com/spf13/cobra"
)

var deleteBranchCmd = &cobra.Command{
	Use:     "delete-branch",
	Aliases: []string{""},
	Short:   "Delete a plan branch",
	Run:     deleteBranch,
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

	branch := strings.TrimSpace(args[0])

	if branch == "main" {
		fmt.Println("🚨 Cannot delete main branch")
		return
	}

	branches, apiErr := api.Client.ListBranches(lib.CurrentPlanId)

	if apiErr != nil {
		fmt.Println("Error getting branches:", apiErr)
		return
	}

	if branch == "" {
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
			fmt.Println("Error selecting branch:", err)
			return
		}

		branch = sel
	}

	found := false
	for _, b := range branches {
		if b.Name == branch {
			found = true
			break
		}
	}

	if !found {
		fmt.Printf("🤷‍♂️ Branch '%s' does not exist\n", branch)
		return
	}

	err := api.Client.DeleteBranch(lib.CurrentPlanId, branch)

	if err != nil {
		fmt.Println("Error deleting branch:", err)
		return
	}

	fmt.Printf("✅ Deleted branch '%s'\n", branch)

	fmt.Println()
	term.PrintCmds("", "branches")
}
