package plan_exec

import (
	"fmt"
	"log"
	"os"
	"plandex-cli/api"
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
		resetAttempts := false

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
			const (
				DebugAndRetry          = "Debug and retry once"
				DebugInFullAutoMode    = "Debug in full auto mode"
				RollbackChangesAndExit = "Rollback changes and exit"
				ApplyChangesAndExit    = "Apply changes and exit"
			)
			opts := []string{
				DebugAndRetry,
				DebugInFullAutoMode,
				RollbackChangesAndExit,
				ApplyChangesAndExit,
			}

			selection, err := term.SelectFromList("What do you want to do?", opts)
			if err != nil {
				term.OutputErrorAndExit("failed to get confirmation user input: %s", err)
			}

			switch selection {
			case DebugAndRetry:
				proceed = true
			case DebugInFullAutoMode:
				proceed = true
				resetAttempts = true

				term.StartSpinner("")
				config, apiErr := api.Client.GetPlanConfig(lib.CurrentPlanId)

				if apiErr != nil {
					term.OutputErrorAndExit("failed to get plan config: %s", apiErr)
				}

				if config.AutoMode != shared.AutoModeFull {
					config.SetAutoMode(shared.AutoModeFull)
					apiErr = api.Client.UpdatePlanConfig(lib.CurrentPlanId, shared.UpdatePlanConfigRequest{
						Config: config,
					})

					if apiErr != nil {
						term.OutputErrorAndExit("failed to update plan config: %s", apiErr)
					}

					applyFlags.AutoCommit = true
					applyFlags.AutoConfirm = true
					applyFlags.AutoExec = true
					applyFlags.AutoDebug = config.AutoDebugTries

					tellFlags.AutoApply = true
					tellFlags.AutoContext = true
					tellFlags.ExecEnabled = true
					tellFlags.SmartContext = true

					term.StopSpinner()
					fmt.Println()
					fmt.Println("✅ Full auto mode enabled")
					fmt.Println()
				} else {
					term.StopSpinner()
				}

			case RollbackChangesAndExit:
				if toRollback != nil {
					lib.Rollback(toRollback, true)
				}
				os.Exit(1)
			case ApplyChangesAndExit:
				onSuccess()
				return
			}
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
				CheckOutdatedContext: func(maybeContexts []*shared.Context, projectPaths *types.ProjectPaths) (bool, bool, error) {
					return lib.CheckOutdatedContextWithOutput(true, true, maybeContexts, projectPaths)
				},
			}, prompt, tellFlags)

			log.Printf("Applying plan after tell")

			if resetAttempts {
				attempt = 0
			}

			lib.MustApplyPlanAttempt(lib.ApplyPlanParams{
				PlanId:      lib.CurrentPlanId,
				Branch:      lib.CurrentBranch,
				ApplyFlags:  applyFlags,
				TellFlags:   tellFlags,
				OnExecFail:  onExecFail,
				ExecCommand: execCommand,
			}, attempt+1)
		}
	}

	return onExecFail
}
