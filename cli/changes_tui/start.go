package changes_tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

func StartChangesUI() error {
	initial := initialModel()

	if len(initial.currentPlan.SortedPaths) == 0 {
		fmt.Println("🤷‍♂️ No changes to apply")
		return nil
	}

	_, err := tea.NewProgram(initial, tea.WithAltScreen()).Run()

	if err != nil {
		return fmt.Errorf("error running changes UI: %v", err)
	}

	// fmt.Println("exited changes UI")
	// spew.Dump(m)

	return nil
}
