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
	Aliases: []string{"c"},
	Short:   "Chat without making changes",
	// Long:  ``,
	Args: cobra.RangeArgs(0, 1),
	Run:  doChat,
}

func init() {
	RootCmd.AddCommand(chatCmd)

	chatCmd.Flags().StringVarP(&tellPromptFile, "file", "f", "", "File containing prompt")
	chatCmd.Flags().BoolVarP(&autoConfirm, "yes", "y", false, "Automatically confirm context updates")
}

func doChat(cmd *cobra.Command, args []string) {
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
			return lib.CheckOutdatedContextWithOutput(false, autoConfirm, maybeContexts)
		},
	}, prompt, false, false, false, false, false, true)
}
