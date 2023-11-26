package changes_tui

import (
	"fmt"
	"plandex/lib"

	tea "github.com/charmbracelet/bubbletea"
)

func StartChangesUI() error {
	initial := initialModel()

	if len(initial.currentPlan.SortedPaths) == 0 {
		fmt.Println("🤷‍♂️ No changes pending")
		return nil
	}

	m, err := tea.NewProgram(initial, tea.WithAltScreen()).Run()

	if err != nil {
		return fmt.Errorf("error running changes UI: %v", err)
	}

	if m.(changesUIModel).shouldApplyAll {
		err := lib.ApplyPlanWithOutput(lib.CurrentPlanName, false)

		if err != nil {
			return fmt.Errorf("error applying plan: %v", err)
		}
	} else if m.(changesUIModel).shouldRejectAll {
		err := lib.DropChangesWithOutput(lib.CurrentPlanName)

		if err != nil {
			return fmt.Errorf("error rejecting plan: %v", err)
		}

	}

	return nil
}
