package lib

import (
	"fmt"
	"os"
	"plandex/api"
	"plandex/term"

	"github.com/fatih/color"
)

func checkContextConflicts(filesByPath map[string]string) (bool, error) {
	// log.Println("Checking for context conflicts.")
	// log.Println(spew.Sdump(filesByPath))

	currentPlan, err := api.Client.GetCurrentPlanState(CurrentPlanId, CurrentBranch)

	if err != nil {
		return false, fmt.Errorf("error getting current plan state: %v", err)
	}

	conflictedPaths := currentPlan.PlanResult.FileResultsByPath.ConflictedPaths(filesByPath)

	// log.Println("Conflicted paths:", conflictedPaths)

	if len(conflictedPaths) > 0 {
		term.StopSpinner()
		color.New(color.Bold, term.ColorHiYellow).Println("⚠️  Some updates conflict with pending changes:")
		for path := range conflictedPaths {
			fmt.Println("📄 " + path)
		}

		fmt.Println()

		res, err := term.ConfirmYesNo("Update context and rebuild changes?")

		if err != nil {
			return false, fmt.Errorf("error confirming update and rebuild: %v", err)
		}

		if !res {
			fmt.Println("Context update canceled")
			os.Exit(0)
		}
	}

	return len(conflictedPaths) > 0, nil
}
