package cmd

import (
	"plandex/auth"
	"plandex/lib"
	"plandex/plan_exec"
	"plandex/term"

	"github.com/plandex/plandex/shared"
	"github.com/spf13/cobra"
)

var continueCmd = &cobra.Command{
	Use:     "continue",
	Aliases: []string{"c"},
	Short:   "Continue the plan",
	Run:     doContinue,
}

func init() {
	RootCmd.AddCommand(continueCmd)

	continueCmd.Flags().BoolVarP(&tellStop, "stop", "s", false, "Stop after a single reply")
	continueCmd.Flags().BoolVarP(&tellNoBuild, "no-build", "n", false, "Don't build files")
	continueCmd.Flags().BoolVar(&tellBg, "bg", false, "Execute autonomously in the background")

	continueCmd.Flags().BoolVarP(&autoConfirm, "yes", "y", false, "Automatically confirm context updates")
	continueCmd.Flags().BoolVarP(&tellAutoApply, "apply", "a", false, "Automatically apply changes (and confirm context updates)")
	continueCmd.Flags().BoolVarP(&autoCommit, "commit", "c", false, "Commit changes to git when --apply/-a is passed")
	continueCmd.Flags().BoolVar(&tellAutoContext, "auto-context", false, "Load and manage context automatically")
	continueCmd.Flags().BoolVar(&noExec, "no-exec", false, "Disable command execution")
	continueCmd.Flags().BoolVar(&autoExec, "auto-exec", false, "Automatically execute commands without confirmation when --apply is passed")
	continueCmd.Flags().Var(newAutoDebugValue(&autoDebug), "debug", "Automatically execute and debug failing commands (optionally specify number of tries—default is 5)")
}

func doContinue(cmd *cobra.Command, args []string) {
	validateTellFlags()

	auth.MustResolveAuthWithOrg()
	lib.MustResolveProject()

	if lib.CurrentPlanId == "" {
		term.OutputNoCurrentPlanErrorAndExit()
	}

	var apiKeys map[string]string
	if !auth.Current.IntegratedModelsMode {
		apiKeys = lib.MustVerifyApiKeys()
	}

	plan_exec.TellPlan(plan_exec.ExecParams{
		CurrentPlanId: lib.CurrentPlanId,
		CurrentBranch: lib.CurrentBranch,
		ApiKeys:       apiKeys,
		CheckOutdatedContext: func(maybeContexts []*shared.Context) (bool, bool, error) {
			return lib.CheckOutdatedContextWithOutput(false, autoConfirm || tellAutoApply || tellAutoContext, maybeContexts)
		},
	}, "", plan_exec.TellFlags{
		TellBg:         tellBg,
		TellStop:       tellStop,
		TellNoBuild:    tellNoBuild,
		IsUserContinue: true,
		ExecEnabled:    !noExec,
		AutoContext:    tellAutoContext,
	})

	if tellAutoApply {
		flags := lib.ApplyFlags{
			AutoConfirm: true,
			AutoCommit:  autoCommit,
			NoCommit:    !autoCommit,
			AutoExec:    autoExec,
			NoExec:      noExec,
			AutoDebug:   autoDebug,
		}

		lib.MustApplyPlan(
			lib.CurrentPlanId,
			lib.CurrentBranch,
			flags,
			plan_exec.GetOnApplyExecFail(flags),
		)
	}
}
