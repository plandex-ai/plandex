package lib

import (
	"fmt"
	"plandex/api"
	"plandex/term"

	"github.com/plandex/plandex/shared"
)

func SelectActiveStream(args []string) (string, string, bool) {
	term.StartSpinner("")
	res, apiErr := api.Client.ListPlansRunning([]string{CurrentProjectId}, false)
	term.StopSpinner()

	if apiErr != nil {
		term.OutputErrorAndExit("Error getting running plans: %v", apiErr)
		return "", "", false
	}

	if len(res.Branches) == 0 {
		fmt.Println("🤷‍♂️ No active plan stream")
		fmt.Println()
		term.PrintCmds("", "ps")
		return "", "", false
	}

	var planId string
	var branch string
	var streamIdOrPlan string

	if len(args) > 0 {
		streamIdOrPlan = args[0]
	}

	if streamIdOrPlan != "" {
		for _, b := range res.Branches {
			id := res.StreamIdByBranchId[b.Id]
			plan := res.PlansById[b.PlanId]
			if id == streamIdOrPlan || plan.Name == streamIdOrPlan {
				planId = b.PlanId
				branch = b.Name
				break
			}
		}
	}

	if planId == "" {
		if len(res.PlansById) == 1 {
			for _, p := range res.PlansById {
				if p.Id == CurrentPlanId {
					planId = p.Id
					break
				}
			}
		}

		if planId == "" {
			var opts []string
			addedPlans := make(map[string]bool)

			for _, plan := range res.PlansById {
				if addedPlans[plan.Id] {
					continue
				}
				opts = append(opts, plan.Name)
				addedPlans[plan.Id] = true
			}

			selected, err := term.SelectFromList("Select an active plan", opts)

			if err != nil {
				term.OutputErrorAndExit("Error selecting plan: %v", err)
			}

			for _, p := range res.PlansById {
				if p.Name == selected {
					planId = p.Id
					break
				}
			}
		}

	}

	if planId == "" {
		fmt.Println("🤷‍♂️ No active plan stream")
		fmt.Println()
		term.PrintCmds("", "ps")

		return "", "", false
	}

	var planBranches []*shared.Branch
	for _, b := range res.Branches {
		if b.PlanId == planId {
			planBranches = append(planBranches, b)
		}
	}

	if len(planBranches) == 0 {
		fmt.Println("🤷‍♂️ No active plan stream")
		fmt.Println()
		term.PrintCmds("", "ps")

		return "", "", false
	}

	if len(args) > 1 {
		maybeBranch := args[1]
		for _, b := range planBranches {
			if b.Name == maybeBranch {
				branch = maybeBranch
				break
			}
		}
	}

	if branch == "" {
		if len(planBranches) == 1 {
			name := planBranches[0].Name
			if name == CurrentBranch {
				branch = name
			}
		}

		if branch == "" {
			opts := make([]string, len(planBranches))

			for i, b := range planBranches {
				opts[i] = b.Name
			}

			selected, err := term.SelectFromList("Select a branch", opts)

			if err != nil {
				term.OutputErrorAndExit("Error selecting branch: %v", err)
			}

			branch = selected
		}
	}

	if branch == "" {
		fmt.Println("🤷‍♂️ No active plan stream")
		fmt.Println()
		term.PrintCmds("", "ps")

		return "", "", false
	}

	return planId, branch, true
}
