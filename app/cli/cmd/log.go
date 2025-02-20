package cmd

import (
	"fmt"
	"plandex-cli/api"
	"plandex-cli/auth"
	"plandex-cli/lib"
	"plandex-cli/term"
	"time"

	"github.com/spf13/cobra"
)

// logCmd represents the log command
var logCmd = &cobra.Command{
	Use:     "log",
	Aliases: []string{"history", "logs"},
	Short:   "Show plan history",
	Long:    `Show plan history`,
	Args:    cobra.NoArgs,
	Run:     runLog,
}

func init() {
	// Add log command
	RootCmd.AddCommand(logCmd)
}

func runLog(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()
	lib.MustResolveProject()

	if lib.CurrentPlanId == "" {
		term.OutputNoCurrentPlanErrorAndExit()
	}

	term.StartSpinner("")
	res, apiErr := api.Client.ListLogs(lib.CurrentPlanId, lib.CurrentBranch)
	term.StopSpinner()

	if apiErr != nil {
		term.OutputErrorAndExit("Error getting logs: %v", apiErr)
	}

	withLocalTimestamps, err := convertTimestampsToLocal(res.Body)

	if err != nil {
		term.OutputErrorAndExit("Error converting timestamps: %v", err)
	}

	term.PageOutput(withLocalTimestamps)

	fmt.Println()
	term.PrintCmds("", "rewind", "continue", "convo", "convo 1", "convo 2-5")

}

func convertTimestampsToLocal(input string) (string, error) {
	t := time.Now()
	zone, _ := t.Zone()
	re := lib.GitLogTimestampRegex

	// Function to convert matched timestamps assuming they are in UTC to local time.
	replaceFunc := func(match string) string {
		t, err := time.Parse(lib.GitLogTimestampFormat, match)
		if err != nil {
			// In case of an error, return the original match.
			return match
		}

		localDt := t.Local()
		formattedTs := localDt.Format("Mon Jan 2, 2006 | 3:04:05pm")

		if localDt.Day() == time.Now().Day() {
			formattedTs = localDt.Format("Today | 3:04:05pm")
		} else if localDt.Day() == time.Now().AddDate(0, 0, -1).Day() {
			formattedTs = localDt.Format("Yesterday | 3:04:05pm")
		}

		// Convert to local time and format back to a string without the timezone to match the original format.
		return formattedTs + " " + zone
	}

	// Find all matches and replace them.
	result := re.ReplaceAllStringFunc(input, replaceFunc)

	return result, nil
}
