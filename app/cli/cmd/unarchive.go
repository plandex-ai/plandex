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
	"github.com/plandex/plandex/shared"
	"github.com/spf13/cobra"
)

var unarchiveCmd = &cobra.Command{
	Use:     "unarchive [name-or-index]",
	Aliases: []string{"unarc"},
	Short:   "Unarchive a plan",
	Args:    cobra.MaximumNArgs(1),
	Run:     unarchive,
}

func init() {
	RootCmd.AddCommand(unarchiveCmd)
}

func unarchive(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()
	lib.MustResolveProject()

	var nameOrIdx string
	if len(args) > 0 {
		nameOrIdx = strings.TrimSpace(args[0])
	}

	var plan *shared.Plan

	term.StartSpinner("")
	plans, apiErr := api.Client.ListArchivedPlans([]string{lib.CurrentProjectId})
	term.StopSpinner()

	if apiErr != nil {
		term.OutputErrorAndExit("Error getting archived plans: %v", apiErr)
	}

	if len(plans) == 0 {
		fmt.Println("🤷‍♂️ No archived plans")
		return
	}

	if nameOrIdx == "" {
		opts := make([]string, len(plans))
		for i, p := range plans {
			opts[i] = p.Name
		}

		selected, err := term.SelectFromList("Select a plan:", opts)
		if err != nil {
			term.OutputErrorAndExit("Error selecting plan: %v", err)
		}

		for _, p := range plans {
			if p.Name == selected {
				plan = p
				break
			}
		}
	} else {
		idx, err := strconv.Atoi(nameOrIdx)
		if err == nil && idx > 0 && idx <= len(plans) {
			plan = plans[idx-1]
		} else {
			for _, p := range plans {
				if p.Name == nameOrIdx {
					plan = p
					break
				}
			}
		}
	}

	if plan == nil {
		term.OutputErrorAndExit("Plan not found")
	}

	err := api.Client.UnarchivePlan(plan.Id)
	if err != nil {
		term.OutputErrorAndExit("Error unarchiving plan: %v", err)
	}

	fmt.Printf("✅ Plan %s unarchived\n", color.New(color.Bold, term.ColorHiGreen).Sprint(plan.Name))

	fmt.Println()
	term.PrintCmds("", "plans", "current")
}
