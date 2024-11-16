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
}

func build(cmd *cobra.Command, args []string) {
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

	didBuild, err := plan_exec.Build(plan_exec.ExecParams{
		CurrentPlanId: lib.CurrentPlanId,
		CurrentBranch: lib.CurrentBranch,
		ApiKeys:       apiKeys,
		CheckOutdatedContext: func(maybeContexts []*shared.Context) (bool, bool, error) {
			return lib.CheckOutdatedContextWithOutput(false, autoConfirm, maybeContexts)
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
	} else if tellAutoApply {
		lib.MustApplyPlan(lib.CurrentPlanId, lib.CurrentBranch, true, autoCommit, !autoCommit)
	} else {
		fmt.Println()
		term.PrintCmds("", "diff", "diff --ui", "apply", "reject", "log")
	}
}
