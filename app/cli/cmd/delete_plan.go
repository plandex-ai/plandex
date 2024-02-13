package cmd

import (
	"fmt"
	"os"
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
			fmt.Fprintln(os.Stderr, "🚨 Can't use both --all and a plan name or index")
			return
		}
	}
	var plan *shared.Plan

	plans, apiErr := api.Client.ListPlans(lib.CurrentProjectId)

	if apiErr != nil {
		fmt.Fprintln(os.Stderr, "Error getting plans:", apiErr.Msg)
		os.Exit(1)
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
			fmt.Fprintln(os.Stderr, "Error selecting plan:", err)
			return
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
				fmt.Fprintln(os.Stderr, "Error: index out of range")
				os.Exit(1)
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
		fmt.Fprintln(os.Stderr, "🚨 Plan not found")
		os.Exit(1)
	}

	apiErr = api.Client.DeletePlan(plan.Id)

	if apiErr != nil {
		fmt.Fprintln(os.Stderr, "Error deleting plan:", apiErr.Msg)
		return
	}

	if lib.CurrentPlanId == plan.Id {
		err := lib.ClearCurrentPlan()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error clearing current plan:", err)
			return
		}
	}

	fmt.Printf("✅ Deleted plan %s\n", color.New(color.Bold).Sprint(plan.Name))
}

func delAll() {
	err := api.Client.DeleteAllPlans(lib.CurrentProjectId)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error deleting all  plans:", err)
		return
	}

	fmt.Fprintln(os.Stderr, "✅ Deleted all plans")
}
