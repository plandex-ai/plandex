package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"plandex/api"
	"plandex/auth"
	"plandex/lib"
	"plandex/term"

	"github.com/fatih/color"
	"github.com/plandex/plandex/shared"
	"github.com/spf13/cobra"
)

var all bool

func init() {
	rmCmd.Flags().BoolVar(&all, "all", false, "Delete all plans")
	RootCmd.AddCommand(rmCmd)
}

// rmCmd represents the rm command
var rmCmd = &cobra.Command{
	Use:     "delete-plan [name-or-index]",
	Aliases: []string{"dp"},
	Short:   "Delete a plan by name or index, or delete all plans with --all flag",
	Args:    cobra.RangeArgs(0, 1),
	Run:     del,
}

func del(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()
	lib.MustResolveProject()

	if all {
		delAll()
		return
	}

	var nameOrIdx string
	if len(args) > 0 {
		nameOrIdx = strings.TrimSpace(args[0])

		if all {
			term.OutputErrorAndExit("Can't use both --all and a plan name or index")
		}
	}
	var plan *shared.Plan

	term.StartSpinner("")
	plans, apiErr := api.Client.ListPlans([]string{lib.CurrentProjectId})
	term.StopSpinner()

	if apiErr != nil {
		term.OutputErrorAndExit("Error getting plans: %v", apiErr)
	}

	if len(plans) == 0 {
		fmt.Println("🤷‍♂️ No plans")
		fmt.Println()
		term.PrintCmds("", "new")
		return
	}

	if nameOrIdx == "" {

		opts := make([]string, len(plans))
		for i, plan := range plans {
			opts[i] = plan.Name
		}

		selected, err := term.SelectFromList("Select a plan", opts)

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

		// see if it's an index
		idx, err := strconv.Atoi(nameOrIdx)

		if err == nil {
			if idx > 0 && idx <= len(plans) {
				plan = plans[idx-1]
			} else {
				term.OutputErrorAndExit("Plan index out of range")
			}
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

	term.StartSpinner("")
	apiErr = api.Client.DeletePlan(plan.Id)
	term.StopSpinner()

	if apiErr != nil {
		term.OutputErrorAndExit("Error deleting plan: %s", apiErr.Msg)
	}

	if lib.CurrentPlanId == plan.Id {
		err := lib.ClearCurrentPlan()
		if err != nil {
			term.OutputErrorAndExit("Error clearing current plan: %v", err)
		}
	}

	fmt.Printf("✅ Deleted plan %s\n", color.New(color.Bold, term.ColorHiCyan).Sprint(plan.Name))
}

func delAll() {
	term.StartSpinner("")
	err := api.Client.DeleteAllPlans(lib.CurrentProjectId)
	term.StopSpinner()

	if err != nil {
		term.OutputErrorAndExit("Error deleting all  plans: %v", err)
	}

	fmt.Println("✅ Deleted all plans")
}
