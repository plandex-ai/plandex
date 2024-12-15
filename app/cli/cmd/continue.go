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

	var config *shared.PlanConfig
	if lib.CurrentPlanId != "" {
		term.StartSpinner("")
		var err error
		config, err = api.Client.GetPlanConfig(lib.CurrentPlanId)
		term.StopSpinner()

		if err != nil {
			term.OutputErrorAndExit("Error getting plan config: %v", err)
		}
	} else {
		term.StartSpinner("")
		var err error
		config, err = api.Client.GetDefaultPlanConfig()
		term.StopSpinner()

		if err != nil {
			term.OutputErrorAndExit("Error getting default plan config: %v", err)
		}
	}

	// Override config with flags
	if cmd.Flags().Changed("yes") {
		config.AutoContext = autoConfirm
	}
	if cmd.Flags().Changed("apply") {
		config.AutoApply = tellAutoApply
	}
	if cmd.Flags().Changed("commit") {
		config.AutoCommit = autoCommit
	}
	if cmd.Flags().Changed("auto-context") {
		config.AutoContext = tellAutoContext
	}
	if cmd.Flags().Changed("no-exec") {
		config.NoExec = noExec
	}
	if cmd.Flags().Changed("auto-exec") {
		config.AutoDebug = autoExec
	}
	if cmd.Flags().Changed("debug") {
		config.AutoDebug = true
		config.AutoDebugTries = autoDebug
	}

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
			return lib.CheckOutdatedContextWithOutput(false, config.AutoContext, maybeContexts)
		},
	}, "", plan_exec.TellFlags{
		TellBg:         tellBg,
		TellStop:       tellStop,
		TellNoBuild:    tellNoBuild,
		IsUserContinue: true,
		ExecEnabled:    !config.NoExec,
		AutoContext:    config.AutoContext,
	})

	if config.AutoApply {
		flags := lib.ApplyFlags{
			AutoConfirm: true,
			AutoCommit:  config.AutoCommit,
			NoCommit:    !config.AutoCommit,
			AutoExec:    config.AutoDebug,
			NoExec:      config.NoExec,
			AutoDebug:   config.AutoDebugTries,
		}

		lib.MustApplyPlan(
			lib.CurrentPlanId,
			lib.CurrentBranch,
			flags,
			plan_exec.GetOnApplyExecFail(flags),
		)
	}
}

