package cmd

import (
	"fmt"
	"os"
	"plandex/auth"
	"plandex/lib"
	"plandex/tell"
	"plandex/term"

	"github.com/spf13/cobra"
)

// Variables to be used in the nextCmd
const continuePrompt = "Continue the plan."

// nextCmd represents the prompt command
var nextCmd = &cobra.Command{
	Use:     "continue",
	Aliases: []string{"c"},
	Short:   "Continue the plan.",
	Run:     doContinue,
}

func init() {
	RootCmd.AddCommand(nextCmd)

	nextCmd.Flags().BoolVarP(&tellStop, "stop", "s", false, "Stop after a single reply")
	nextCmd.Flags().BoolVarP(&tellNoBuild, "no-build", "n", false, "Don't build files")
}

func doContinue(cmd *cobra.Command, args []string) {
	if os.Getenv("OPENAI_API_KEY") == "" {
		term.OutputNoApiKeyMsg()
		os.Exit(1)
	}

	auth.MustResolveAuthWithOrg()
	lib.MustResolveProject()
	lib.MustCheckOutdatedContextWithOutput()

	if lib.CurrentPlanId == "" {
		fmt.Fprintln(os.Stderr, "No current plan")
		return
	}

	tell.TellPlan(continuePrompt, tellBg, tellStop, tellNoBuild)
}
