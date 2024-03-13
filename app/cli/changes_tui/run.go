package changes_tui

import (
	"fmt"
	"plandex/api"
	"plandex/lib"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/plandex/plandex/shared"
)

func StartChangesUI(currentPlan *shared.CurrentPlanState) error {
	initial := initialModel(currentPlan)

	if len(initial.currentPlan.PlanResult.SortedPaths) == 0 {
		fmt.Println("🤷‍♂️ No changes pending")
		return nil
	}

	m, err := tea.NewProgram(initial, tea.WithAltScreen()).Run()

	if err != nil {
		return fmt.Errorf("error running changes UI: %v", err)
	}

	if m.(changesUIModel).shouldApplyAll {
		lib.MustApplyPlan(lib.CurrentPlanId, lib.CurrentBranch, false)
	} else if m.(changesUIModel).shouldRejectAll {
		err := api.Client.RejectAllChanges(lib.CurrentPlanId, lib.CurrentBranch)

		if err != nil {
			return fmt.Errorf("error rejecting plan: %v", err)
		}

	}

	return nil
}
