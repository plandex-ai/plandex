package cmd

import (
	"fmt"
	"path"
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

func matchPlansByPattern(pattern string, plans []*shared.Plan) []*shared.Plan {
	var matched []*shared.Plan
	for _, plan := range plans {
		if isMatched, err := path.Match(pattern, plan.Name); err == nil && isMatched {
			matched = append(matched, plan)
		}
	}
	return matched
}

func del(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()
	lib.MustResolveProject()

	if all {
		delAll()
		return
	}

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

	var plansToDelete []*shared.Plan

	if len(args) == 0 {
		// Interactive selection
		opts := make([]string, len(plans))
		for i, plan := range plans {
			opts[i] = plan.Name
		}

		selected, err := term.SelectFromList("Select a plan:", opts)
		if err != nil {
			term.OutputErrorAndExit("Error selecting plan: %v", err)
		}

		for _, p := range plans {
			if p.Name == selected {
				plansToDelete = append(plansToDelete, p)
				break
			}
		}
	} else {
		nameOrPattern := strings.TrimSpace(args[0])

		// Check if it's a range of indices
		if strings.Contains(nameOrPattern, "-") {
			// Create single-element slice with the range pattern
			rangeArgs := []string{nameOrPattern}
			indices := parseIndices(rangeArgs)
			for idx := range indices {
				if idx >= 0 && idx < len(plans) {
					plansToDelete = append(plansToDelete, plans[idx])
				}
			}
		} else if strings.Contains(nameOrPattern, "*") {
			// Wildcard pattern matching
			plansToDelete = matchPlansByPattern(nameOrPattern, plans)
		} else {
			// Try as index first
			idx, err := strconv.Atoi(nameOrPattern)
			if err == nil {
				if idx > 0 && idx <= len(plans) {
					plansToDelete = append(plansToDelete, plans[idx-1])
				} else {
					term.OutputErrorAndExit("Plan index out of range")
				}
			} else {
				// Try exact name match
				for _, p := range plans {
					if p.Name == nameOrPattern {
						plansToDelete = append(plansToDelete, p)
						break
					}
				}
			}
		}
	}

	if len(plansToDelete) == 0 {
		term.OutputErrorAndExit("No matching plans found")
	}

	// Show confirmation with list of plans to be deleted
	fmt.Printf("\nThe following %d plan(s) will be deleted:\n", len(plansToDelete))
	for _, p := range plansToDelete {
		fmt.Printf("  - %s\n", color.New(color.Bold, term.ColorHiCyan).Sprint(p.Name))
	}
	fmt.Println()

	confirmed, err := term.ConfirmYesNo("Are you sure you want to delete these plans?")
	if err != nil {
		term.OutputErrorAndExit("Error getting confirmation: %v", err)
	}
	if !confirmed {
		fmt.Println("Operation cancelled")
		return
	}

	// Delete the plans
	term.StartSpinner("")
	for _, p := range plansToDelete {
		apiErr = api.Client.DeletePlan(p.Id)
		if apiErr != nil {
			term.StopSpinner()
			term.OutputErrorAndExit("Error deleting plan %s: %s", p.Name, apiErr.Msg)
		}

		if lib.CurrentPlanId == p.Id {
			err := lib.ClearCurrentPlan()
			if err != nil {
				term.OutputErrorAndExit("Error clearing current plan: %v", err)
			}
		}
	}
	term.StopSpinner()

	if len(plansToDelete) == 1 {
		fmt.Printf("✅ Deleted plan '%s'\n", plansToDelete[0].Name)
	} else {
		fmt.Printf("✅ Deleted %d plans\n", len(plansToDelete))
	}
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
