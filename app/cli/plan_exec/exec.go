package plan_exec

import (
	"fmt"
	"log"
	"plandex/auth"
	"plandex/lib"
	"plandex/term"

	"github.com/fatih/color"
	"github.com/plandex/plandex/shared"
)

func GetOnApplyExecFail(flags lib.ApplyFlags) func(status int, output string, attempt int) {
	var onExecFail func(status int, output string, attempt int)
	onExecFail = func(status int, output string, attempt int) {
		var proceed bool

		if flags.AutoDebug > 0 {
			if attempt >= flags.AutoDebug {
				color.New(term.ColorHiRed, color.Bold).Printf("Commands failed %d times.\n", attempt)
			} else {
				proceed = true
			}
		}

		if !proceed {
			confirm, err := term.ConfirmYesNo("Auto-debug?")

			if err != nil {
				term.OutputErrorAndExit("failed to get confirmation user input: %s", err)
			}

			proceed = confirm
		}

		if proceed {
			var apiKeys map[string]string
			if !auth.Current.IntegratedModelsMode {
				apiKeys = lib.MustVerifyApiKeysSilent()
			}
			prompt := fmt.Sprintf("Execution of '_apply.sh' failed with exit status %d. Output:\n\n%s\n\n--\n\n",
				status, output)

			TellPlan(ExecParams{
				CurrentPlanId: lib.CurrentPlanId,
				CurrentBranch: lib.CurrentBranch,
				ApiKeys:       apiKeys,
				CheckOutdatedContext: func(maybeContexts []*shared.Context) (bool, bool, error) {
					return lib.CheckOutdatedContextWithOutput(false, true, maybeContexts)
				},
			}, prompt, TellFlags{IsApplyDebug: true, ExecEnabled: true})

			log.Printf("Applying plan after tell")

			lib.MustApplyPlanAttempt(lib.CurrentPlanId, lib.CurrentBranch, flags, onExecFail, attempt+1)
		}
	}

	return onExecFail
}
