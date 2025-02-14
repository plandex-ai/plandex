package plan_exec

import (
	"fmt"
	"log"
	"plandex-cli/auth"
	"plandex-cli/lib"
	"plandex-cli/term"
	"plandex-cli/types"

	shared "plandex-shared"

	"github.com/fatih/color"
)

func GetOnApplyExecFail(applyFlags types.ApplyFlags, tellFlags types.TellFlags) types.OnApplyExecFailFn {
	return getOnApplyExecFail(applyFlags, tellFlags, "")
}

func GetOnApplyExecFailWithCommand(applyFlags types.ApplyFlags, tellFlags types.TellFlags, execCommand string) types.OnApplyExecFailFn {
	return getOnApplyExecFail(applyFlags, tellFlags, execCommand)
}

func getOnApplyExecFail(applyFlags types.ApplyFlags, tellFlags types.TellFlags, execCommand string) types.OnApplyExecFailFn {
	var onExecFail types.OnApplyExecFailFn
	onExecFail = func(status int, output string, attempt int, toRollback *types.ApplyRollbackPlan, onErr types.OnErrFn, onSuccess func()) {
		var proceed bool

		if applyFlags.AutoDebug > 0 {
			if attempt >= applyFlags.AutoDebug {
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
			prompt := fmt.Sprintf("Execution failed with exit status %d. Output:\n\n%s\n\n--\n\n",
				status, output)

			tellFlags.IsUserContinue = false

			if execCommand != "" {
				tellFlags.IsApplyDebug = false
				tellFlags.ExecEnabled = false
				tellFlags.IsUserDebug = true
			} else {
				tellFlags.IsApplyDebug = true
				tellFlags.ExecEnabled = true
				tellFlags.IsUserDebug = false
			}

			log.Printf("Calling TellPlan for next debug attempt")

			TellPlan(ExecParams{
				CurrentPlanId: lib.CurrentPlanId,
				CurrentBranch: lib.CurrentBranch,
				ApiKeys:       apiKeys,
				CheckOutdatedContext: func(maybeContexts []*shared.Context) (bool, bool, error) {
					return lib.CheckOutdatedContextWithOutput(true, true, maybeContexts)
				},
			}, prompt, tellFlags)

			log.Printf("Applying plan after tell")

			lib.MustApplyPlanAttempt(lib.ApplyPlanParams{
				PlanId:      lib.CurrentPlanId,
				Branch:      lib.CurrentBranch,
				ApplyFlags:  applyFlags,
				TellFlags:   tellFlags,
				OnExecFail:  onExecFail,
				ExecCommand: execCommand,
			}, attempt+1)
		} else {
			res, err := term.SelectFromList("Still apply file changes or roll back?", []string{string(types.ApplyRollbackOptionKeep), string(types.ApplyRollbackOptionRollback)})

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
