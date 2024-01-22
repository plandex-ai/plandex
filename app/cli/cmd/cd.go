package cmd

import (
	"fmt"
	"os"
	"plandex/api"
	"plandex/auth"
	"plandex/lib"
	"strconv"
	"time"

	"github.com/plandex/plandex/shared"
	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(cdCmd)
}

var cdCmd = &cobra.Command{
	Use:     "cd [name-or-index]",
	Aliases: []string{"set-plan"},
	Short:   "Set current plan by name or index",
	Args:    cobra.ExactArgs(1),
	Run:     cd,
}

func cd(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()
	lib.MustResolveProject()

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

	err = lib.SetCurrentPlan(plan.Id)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error setting current plan:", err)
		os.Exit(1)
	}

	// fire and forget SetProjectPlan request (we don't care about the response or errors)
	// this only matters for setting the current plan on a new device (i.e. when the current plan is not set)
	go api.Client.SetProjectPlan(lib.CurrentProjectId, shared.SetProjectPlanRequest{PlanId: plan.Id})

	// give the SetProjectPlan request some time to be sent before exiting
	time.Sleep(50 * time.Millisecond)

	fmt.Fprintln(os.Stderr, "✅ Changed current plan to "+name)
}
