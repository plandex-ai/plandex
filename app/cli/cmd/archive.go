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
	"github.com/spf13/cobra"
)

var archiveCmd = &cobra.Command{
	Use:     "archive [name-or-index]",
	Aliases: []string{"ar"},
	Short:   "Archive a plan",
	Args:    cobra.MaximumNArgs(1),
	Run:     archive,
}

func init() {
	RootCmd.AddCommand(archiveCmd)
}

func archive(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()
	lib.MustResolveProject()

	var nameOrIdx string
	if len(args) > 0 {
		nameOrIdx = strings.TrimSpace(args[0])
	}

	var plan *shared.Plan

	term.StartSpinner("Loading plans")
	plans, apiErr := api.Client.ListPlans([]string{lib.CurrentProjectId})
	term.StopSpinner()

	if apiErr != nil {
		term.OutputErrorAndExit("Error getting plans: %v", apiErr)
	}

	if len(plans) == 0 {
		fmt.Println("🤷‍♂️ No plans available to archive")
		return
	}

	if nameOrIdx == "" {
		opts := make([]string, len(plans))
		for i, p := range plans {
			opts[i] = p.Name
		}

		selected, err := term.SelectFromList("Select a plan to archive", opts)
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

	err := api.Client.ArchivePlan(plan.Id)
	if err != nil {
		term.OutputErrorAndExit("Error archiving plan: %v", err)
	}

	fmt.Printf("✅ Plan %s archived successfully\n", color.New(color.Bold).Sprint(plan.Name))
}
