package changes_tui

import (
	"fmt"
	"plandex-cli/lib"
	"plandex-cli/plan_exec"
	"plandex-cli/term"

	shared "plandex-shared"

	tea "github.com/charmbracelet/bubbletea"
)

var program *tea.Program

func StartChangesUI(currentPlan *shared.CurrentPlanState) error {
	initial := initialModel(currentPlan)

	if len(initial.currentPlan.PlanResult.SortedPaths) == 0 {
		fmt.Println("🤷‍♂️ No changes pending")
		return nil
	}

	program = tea.NewProgram(initial, tea.WithAltScreen())

	m, err := program.Run()

	if err != nil {
		return fmt.Errorf("error running changes UI: %v", err)
	}

	var mod *changesUIModel
	c, ok := m.(*changesUIModel)

	if ok {
		mod = c
	} else {
		c := m.(changesUIModel)
		mod = &c
	}

	if mod.shouldApplyAll {
		flags := lib.ApplyFlags{
			AutoConfirm: true,
		}
		lib.MustApplyPlan(
			lib.CurrentPlanId,
			lib.CurrentBranch,
			flags,
			plan_exec.GetOnApplyExecFail(flags),
		)
	}

	if mod.rejectFileErr != nil {
		fmt.Println()
		term.OutputErrorAndExit("Server error: " + mod.rejectFileErr.Msg)
	}

	if mod.justRejectedFile && len(mod.currentPlan.PlanResult.SortedPaths) == 0 {
		fmt.Println("🚫 All changes rejected")
		return nil
	}

	return nil
}
