package cmd

import (
	"fmt"
	"os"
	"strconv"

	"plandex/api"
	"plandex/auth"
	"plandex/lib"

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
	Aliases: []string{"del"},
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

	nameOrIdx := args[0]
	var plan *shared.Plan

	plans, apiErr := api.Client.ListPlans(lib.CurrentProjectId)

	if apiErr != nil {
		fmt.Fprintln(os.Stderr, "Error getting plans:", apiErr.Msg)
		os.Exit(1)
	}

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

		if plan == nil {
			fmt.Fprintln(os.Stderr, "Error: plan not found")
			os.Exit(1)
		}
	}

	apiErr = api.Client.DeletePlan(plan.Id)

	if apiErr != nil {
		fmt.Fprintln(os.Stderr, "Error deleting plan:", apiErr.Msg)
		return
	}

	if lib.CurrentPlanId == plan.Id {
		err = lib.ClearCurrentPlan()
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
