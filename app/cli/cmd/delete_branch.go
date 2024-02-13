package cmd

import (
	"fmt"
	"os"
	"plandex/api"
	"plandex/auth"
	"plandex/lib"
	"plandex/term"
	"strconv"
	"strings"

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

	branches, apiErr := api.Client.ListBranches(lib.CurrentPlanId)

	if apiErr != nil {
		fmt.Println("Error getting branches:", apiErr)
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
			fmt.Println("Error selecting branch:", err)
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
			fmt.Fprintln(os.Stderr, "Error: index out of range")
			os.Exit(1)
		}
	} else {
		for _, b := range branches {
			if b.Name == nameOrIdx {
				branch = b.Name
				break
			}
		}

		if branch == "" {
			fmt.Fprintln(os.Stderr, "Error: branch not found")
			os.Exit(1)
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
		fmt.Printf("🤷‍♂️ Branch '%s' does not exist\n", branch)
		return
	}

	apiErr = api.Client.DeleteBranch(lib.CurrentPlanId, branch)

	if apiErr != nil {
		fmt.Println("Error deleting branch:", apiErr)
		return
	}

	fmt.Printf("✅ Deleted branch '%s'\n", branch)

	fmt.Println()
	term.PrintCmds("", "branches")
}
