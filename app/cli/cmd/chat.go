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

var chatCmd = &cobra.Command{
	Use:     "chat [prompt]",
	Aliases: []string{"ct"},
	Short:   "Chat without making changes",
	// Long:  ``,
	Args: cobra.RangeArgs(0, 1),
	Run:  doChat,
}

func init() {
	RootCmd.AddCommand(chatCmd)

	chatCmd.Flags().StringVarP(&tellPromptFile, "file", "f", "", "File containing prompt")
	chatCmd.Flags().BoolVarP(&autoConfirm, "yes", "y", false, "Automatically confirm context updates")
	chatCmd.Flags().BoolVar(&tellAutoContext, "auto-context", false, "Load and manage context automatically")
}

func doChat(cmd *cobra.Command, args []string) {
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
	if cmd.Flags().Changed("auto-context") {
		config.AutoContext = tellAutoContext
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

	prompt := getTellPrompt(args)

	if prompt == "" {
		fmt.Println("🤷‍♂️ No prompt to send")
		return
	}

	plan_exec.TellPlan(plan_exec.ExecParams{
		CurrentPlanId: lib.CurrentPlanId,
		CurrentBranch: lib.CurrentBranch,
		ApiKeys:       apiKeys,
		CheckOutdatedContext: func(maybeContexts []*shared.Context) (bool, bool, error) {
			return lib.CheckOutdatedContextWithOutput(false, config.AutoContext, maybeContexts)
		},
	}, prompt, plan_exec.TellFlags{
		IsChatOnly:  true,
		AutoContext: config.AutoContext,
	})
}

