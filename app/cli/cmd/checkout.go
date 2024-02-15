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

	"github.com/fatih/color"
	"github.com/plandex/plandex/shared"
	"github.com/spf13/cobra"
)

const (
	OptCreateNewBranch = "Create a new branch"
)

var checkoutCmd = &cobra.Command{
	Use:     "checkout [name-or-index]",
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

	branchName := ""
	willCreate := false

	var nameOrIdx string
	if len(args) > 0 {
		nameOrIdx = strings.TrimSpace(args[0])
	}

	branches, apiErr := api.Client.ListBranches(lib.CurrentPlanId)

	if apiErr != nil {
		fmt.Println("Error getting branches:", apiErr)
		return
	}

	if nameOrIdx != "" {
		idx, err := strconv.Atoi(nameOrIdx)

		if err == nil {
			if idx > 0 && idx <= len(branches) {
				branchName = branches[idx-1].Name
			} else {
				fmt.Fprintf(os.Stderr, "🚨 Branch %d not found\n", idx)
				os.Exit(1)
			}
		} else {
			for _, b := range branches {
				if b.Name == nameOrIdx {
					branchName = b.Name
					break
				}
			}
		}

		if branchName == "" {
			fmt.Printf("🤷‍♂️ Branch %s not found\n", color.New(color.Bold, color.FgHiCyan).Sprint(nameOrIdx))
			res, err := term.ConfirmYesNo("Create it now?")

			if err != nil {
				fmt.Println("Error getting user input:", err)
			}

			if res {
				branchName = nameOrIdx
				willCreate = true
			} else {
				return
			}
		}

	}

	if nameOrIdx == "" {
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
		fmt.Fprintln(os.Stderr, "🚨 Branch not found")
		os.Exit(1)
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
		// fmt.Printf("✅ Created branch %s\n", color.New(color.Bold, color.FgHiGreen).Sprint(branchName))
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
