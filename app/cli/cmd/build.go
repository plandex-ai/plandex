package cmd

import (
	"fmt"
	"plandex/auth"
	"plandex/lib"
	"plandex/plan_exec"
	"plandex/term"

	"github.com/plandex/plandex/shared"
	"github.com/spf13/cobra"
)

var buildCmd = &cobra.Command{
	Use:     "build",
	Aliases: []string{"b"},
	Short:   "Build pending changes",
	// Long:  ``,
	Args: cobra.NoArgs,
	Run:  build,
}

func init() {
	RootCmd.AddCommand(buildCmd)
	buildCmd.Flags().BoolVar(&tellBg, "bg", false, "Execute autonomously in the background")

	buildCmd.Flags().BoolVarP(&autoConfirm, "yes", "y", false, "Automatically confirm context updates")
	buildCmd.Flags().BoolVarP(&tellAutoApply, "apply", "a", false, "Automatically apply changes (and confirm context updates)")
	buildCmd.Flags().BoolVarP(&autoCommit, "commit", "c", false, "Commit changes to git when --apply/-a is passed")
	buildCmd.Flags().BoolVar(&noExec, "no-exec", false, "Disable command execution")
	buildCmd.Flags().BoolVar(&autoExec, "auto-exec", false, "Automatically execute commands without confirmation when --apply is passed")
	buildCmd.Flags().Var(newAutoDebugValue(&autoDebug), "debug", "Automatically execute and debug failing commands (optionally specify number of tries—default is 5)")
}

func build(cmd *cobra.Command, args []string) {
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

	didBuild, err := plan_exec.Build(plan_exec.ExecParams{
		CurrentPlanId: lib.CurrentPlanId,
		CurrentBranch: lib.CurrentBranch,
		ApiKeys:       apiKeys,
		CheckOutdatedContext: func(maybeContexts []*shared.Context) (bool, bool, error) {
			return lib.CheckOutdatedContextWithOutput(false, config.AutoContext, maybeContexts)
		},
	}, tellBg)

	if err != nil {
		term.OutputErrorAndExit("Error building plan: %v", err)
	}

	if !didBuild {
		fmt.Println()
		term.PrintCmds("", "log", "tell", "continue")
		return
	}

	if tellBg {
		fmt.Println("🏗️ Building plan in the background")
		fmt.Println()
		term.PrintCmds("", "ps", "connect", "stop")
	} else if config.AutoApply {
		flags := lib.ApplyFlags{
			AutoConfirm: true,
			AutoCommit:  config.AutoCommit,
			NoCommit:    !config.AutoCommit,
			NoExec:      config.NoExec,
			AutoExec:    config.AutoDebug,
			AutoDebug:   config.AutoDebugTries,
		}

		lib.MustApplyPlan(
			lib.CurrentPlanId,
			lib.CurrentBranch,
			flags,
			plan_exec.GetOnApplyExecFail(flags),
		)
	} else {
		fmt.Println()
		term.PrintCmds("", "diff", "diff --ui", "apply", "reject", "log")
	}
}

