package cmd

import (
	"fmt"
	"plandex-cli/auth"
	"plandex-cli/lib"
	"plandex-cli/plan_exec"
	"plandex-cli/term"
	"plandex-cli/types"

	shared "plandex-shared"

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

	initExecFlags(buildCmd, initExecFlagsParams{
		omitFile:        true,
		omitNoBuild:     true,
		omitEditor:      true,
		omitStop:        true,
		omitAutoContext: true,
	})
}

func build(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()
	lib.MustResolveProject()
	mustSetPlanExecFlags(cmd)

	var apiKeys map[string]string
	if !auth.Current.IntegratedModelsMode {
		apiKeys = lib.MustVerifyApiKeys()
	}

	didBuild, err := plan_exec.Build(plan_exec.ExecParams{
		CurrentPlanId: lib.CurrentPlanId,
		CurrentBranch: lib.CurrentBranch,
		ApiKeys:       apiKeys,
		CheckOutdatedContext: func(maybeContexts []*shared.Context) (bool, bool, error) {
			auto := autoConfirm || tellAutoApply || tellAutoContext
			return lib.CheckOutdatedContextWithOutput(auto, auto, maybeContexts)
		},
	}, types.BuildFlags{
		BuildBg:   tellBg,
		AutoApply: tellAutoApply,
	})

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
		applyFlags := types.ApplyFlags{
			AutoConfirm: true,
			AutoCommit:  autoCommit,
			NoCommit:    !autoCommit,
			NoExec:      noExec,
			AutoExec:    autoExec,
			AutoDebug:   autoDebug,
		}

		tellFlags := types.TellFlags{
			AutoContext: tellAutoContext,
			ExecEnabled: !noExec,
			AutoApply:   tellAutoApply,
		}

		lib.MustApplyPlan(lib.ApplyPlanParams{
			PlanId:     lib.CurrentPlanId,
			Branch:     lib.CurrentBranch,
			ApplyFlags: applyFlags,
			TellFlags:  tellFlags,
			OnExecFail: plan_exec.GetOnApplyExecFail(applyFlags, tellFlags),
		})
	} else {
		fmt.Println()
		term.PrintCmds("", "diff", "diff --ui", "apply", "reject", "log")
	}
}
