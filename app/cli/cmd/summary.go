package cmd

import (
	"fmt"
	"plandex/api"
	"plandex/auth"
	"plandex/lib"
	"plandex/term"

	"github.com/spf13/cobra"
)

var summaryPlain bool

var statusCmd = &cobra.Command{
	Use:   "summary",
	Short: "Show the latest summary of the current plan",
	Run:   status,
}

func init() {
	RootCmd.AddCommand(statusCmd)

	statusCmd.Flags().BoolVarP(&summaryPlain, "plain", "p", false, "Output summary in plain text with no ANSI codes")
}

func status(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()
	lib.MustResolveProject()

	term.StartSpinner("")
	status, apiErr := api.Client.GetPlanStatus(lib.CurrentPlanId, lib.CurrentBranch)
	term.StopSpinner()

	if apiErr != nil {
		term.OutputErrorAndExit("Error loading conversation: %v", apiErr.Msg)
	}

	if status == "" {
		fmt.Println("🤷‍♂️ No summary available")
	}

	if summaryPlain {
		fmt.Println(status)
		return
	}

	md, err := term.GetMarkdown(status)

	if err != nil {
		term.OutputErrorAndExit("Error formatting markdown: %v", err)
	}

	fmt.Println(md)
}
