package cmd

import (
	"fmt"
	"plandex/api"
	"plandex/auth"
	"plandex/lib"
	"plandex/term"
	"regexp"
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
		fmt.Println("No current plan")
		return
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

}

func convertTimestampsToLocal(input string) (string, error) {
	t := time.Now()
	zone, _ := t.Zone()

	parseFmt := "Mon Jan 2, 2006 | 3:04:05pm MST"
	re := regexp.MustCompile(`\w{3} \w{3} \d{1,2}, \d{4} \| \d{1,2}:\d{2}:\d{2}(am|pm) UTC`)

	// Function to convert matched timestamps assuming they are in UTC to local time.
	replaceFunc := func(match string) string {
		t, err := time.Parse(parseFmt, match)
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
