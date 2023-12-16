package cmd

import (
	"fmt"
	"os"
	"plandex/api"
	"plandex/lib"
	"strconv"

	"github.com/plandex/plandex/shared"
	"github.com/spf13/cobra"
)

var autoConfirm bool

func init() {
	applyCmd.Flags().BoolVarP(&autoConfirm, "yes", "y", false, "Automatically confirm unless plan is outdated")
	RootCmd.AddCommand(applyCmd)
}

var applyCmd = &cobra.Command{
	Use:     "apply [name-or-index]",
	Aliases: []string{"ap"},
	Short:   "Apply a plan to the project",
	Args:    cobra.MaximumNArgs(1),
	RunE:    apply,
}

func apply(cmd *cobra.Command, args []string) error {
	lib.MustResolveProject()

	var planId string

	if len(args) == 0 {
		if lib.CurrentPlanId == "" {
			fmt.Fprintln(os.Stderr, "No current plan")
			os.Exit(1)
		}

		planId = lib.CurrentPlanId
	} else {
		nameOrIdx := args[0]

		if nameOrIdx == "current" {
			if lib.CurrentPlanId == "" {
				fmt.Fprintln(os.Stderr, "No current plan")
				os.Exit(1)
			}

			planId = lib.CurrentPlanId
		} else {
			var plan *shared.Plan

			plans, err := api.Client.ListPlans(lib.CurrentProjectId)

			if err != nil {
				fmt.Fprintln(os.Stderr, "Error getting plans:", err)
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
					fmt.Printf("🤷‍♂️ Plan with name '%s' doesn't exist\n", nameOrIdx)
					os.Exit(1)
				}
			}

			planId = plan.Id
		}
	}

	return lib.ApplyPlanWithOutput(planId, autoConfirm)
}
