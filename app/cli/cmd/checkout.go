package cmd

import (
	"fmt"
	"os"
	"plandex/api"
	"plandex/auth"
	"plandex/lib"
	"plandex/term"
	"strings"

	"github.com/fatih/color"
	"github.com/plandex/plandex/shared"
	"github.com/spf13/cobra"
)

const (
	OptCreateNewBranch = "Create a new branch"
)

var checkoutCmd = &cobra.Command{
	Use:     "checkout",
	Aliases: []string{"co"},
	Short:   "Checkout an existing plan branch or create a new one",
	Run:     checkout,
	Args:    cobra.MaximumNArgs(1),
}

func init() {
	RootCmd.AddCommand(checkoutCmd)
}

func checkout(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()
	lib.MustResolveProject()

	if lib.CurrentPlanId == "" {
		fmt.Println("🤷‍♂️ No current plan")
		return
	}

	branches, apiErr := api.Client.ListBranches(lib.CurrentPlanId)

	if apiErr != nil {
		fmt.Println("Error getting branches:", apiErr)
		return
	}

	branchName := ""
	willCreate := false
	if len(args) > 0 {
		branchName = strings.TrimSpace(args[0])
	}

	if branchName == "" {
		opts := make([]string, len(branches))
		for i, branch := range branches {
			opts[i] = branch.Name
		}
		opts = append(opts, OptCreateNewBranch)

		selected, err := term.SelectFromList("Select a branch", opts)

		if err != nil {
			fmt.Println("Error selecting branch:", err)
			return
		}

		if selected == OptCreateNewBranch {
			branchName, err = term.GetUserStringInput("Branch name")
			if err != nil {
				fmt.Println("Error getting branch name:", err)
				return
			}
			willCreate = true
		} else {
			branchName = selected
		}
	}

	if branchName == "" {
		// should never happen
		fmt.Fprintln(os.Stderr, "🚨 Error selecting branch")
		os.Exit(1)
	}

	if !willCreate {
		found := false
		for _, branch := range branches {
			if branch.Name == branchName {
				found = true
				break
			}
		}

		if !found {
			fmt.Printf("🤷‍♂️ Branch '%s' not found\n", branchName)
			res, err := term.ConfirmYesNo("Create it now?")

			if err != nil {
				fmt.Println("Error getting user input:", err)
			}

			if res {
				willCreate = true
			} else {
				return
			}
		}
	}

	if willCreate {
		term.StartSpinner("🌱 Creating branch...")
		err := api.Client.CreateBranch(lib.CurrentPlanId, lib.CurrentBranch, shared.CreateBranchRequest{Name: branchName})

		if err != nil {
			term.StopSpinner()
			fmt.Println("Error creating branch:", err)
			return
		}

		term.StopSpinner()
		fmt.Printf("✅ Created branch %s\n", color.New(color.Bold, color.FgHiGreen).Sprint(branchName))
	}

	err := lib.WriteCurrentBranch(branchName)

	if err != nil {
		fmt.Println("Error setting current branch:", err)
		return
	}

	fmt.Printf("✅ Checked out branch %s\n", color.New(color.Bold, color.FgHiGreen).Sprint(branchName))

	fmt.Println()
	term.PrintCmds("", "load", "tell", "branches", "delete-branch")

}
