package plan_exec

import (
	"fmt"
	"log"
	"plandex/auth"
	"plandex/lib"
	"plandex/term"
	"plandex/types"

	"github.com/fatih/color"
	"github.com/plandex/plandex/shared"
)

func GetOnApplyExecFail(flags lib.ApplyFlags) types.OnApplyExecFailFn {
	var onExecFail types.OnApplyExecFailFn
	onExecFail = func(status int, output string, attempt int, toRollback *types.ApplyRollbackPlan, onErr types.OnErrFn, onSuccess func()) {
		var proceed bool

		if flags.AutoDebug > 0 {
			if attempt >= flags.AutoDebug {
				timesLbl := "times"
				if attempt == 1 {
					timesLbl = "time"
				}
				color.New(term.ColorHiRed, color.Bold).Printf("Commands failed %d %s.\n", attempt, timesLbl)
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
			if toRollback != nil && toRollback.HasChanges() {
				lib.Rollback(toRollback, true)
			}

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
					return lib.CheckOutdatedContextWithOutput(true, true, maybeContexts)
				},
			}, prompt, TellFlags{IsApplyDebug: true, ExecEnabled: true})

			log.Printf("Applying plan after tell")

			lib.MustApplyPlanAttempt(lib.CurrentPlanId, lib.CurrentBranch, flags, onExecFail, attempt+1)
		} else {
			res, err := term.SelectFromList("Still apply other changes or roll back all changes?", []string{string(types.ApplyRollbackOptionKeep), string(types.ApplyRollbackOptionRollback)})

			if err != nil {
				onErr("failed to get rollback confirmation user input: %s", err)
			}

			if res == string(types.ApplyRollbackOptionRollback) {
				lib.Rollback(toRollback, true)
			} else {
				onSuccess()
			}
		}
	}

	return onExecFail
}
