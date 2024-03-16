package cmd

import (
	"fmt"
	"os"
	"plandex/api"
	"plandex/auth"
	"plandex/format"
	"plandex/lib"
	"plandex/term"

	"github.com/olekukonko/tablewriter"
	"github.com/plandex/plandex/shared"
	"github.com/spf13/cobra"
)

var psCmd = &cobra.Command{
	Use:   "ps",
	Short: "List plans with active or recently finished streams",
	Run:   ps,
}

func init() {
	RootCmd.AddCommand(psCmd)
}

func ps(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()
	lib.MustResolveProject()

	if lib.CurrentPlanId == "" {
		fmt.Println("🤷‍♂️ No current plan")
		return
	}

	term.StartSpinner("")
	res, apiErr := api.Client.ListPlansRunning([]string{lib.CurrentProjectId}, true)
	term.StopSpinner()

	if apiErr != nil {
		term.OutputErrorAndExit("Error getting running plans: %v", apiErr)
		return
	}

	if len(res.Branches) == 0 {
		fmt.Println("🤷‍♂️ No active or recently finished streams")
		return
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetAutoWrapText(false)
	table.SetHeader([]string{"Pid", "Plan", "Branch", "Started", "Status"})

	for _, b := range res.Branches {
		id := res.StreamIdByBranchId[b.Id]
		plan := res.PlansById[b.PlanId]

		status := "Active"
		finishedAt := res.StreamFinishedAtByBranchId[b.Id]
		switch b.Status {
		case shared.PlanStatusFinished:
			status = "Finished " + format.Time(finishedAt)
		case shared.PlanStatusError:
			status = "Error " + format.Time(finishedAt)
		case shared.PlanStatusStopped:
			status = "Stopped " + format.Time(finishedAt)
		case shared.PlanStatusMissingFile:
			status = "Missing file"
		}

		row := []string{
			id[:4],
			plan.Name,
			b.Name,
			format.Time(res.StreamStartedAtByBranchId[b.Id]),
			status,
		}

		var style []tablewriter.Colors
		if b.Name == lib.CurrentPlanId {
			style = []tablewriter.Colors{
				{tablewriter.FgGreenColor, tablewriter.Bold},
			}
		} else {
			style = []tablewriter.Colors{
				{tablewriter.Bold},
			}
		}

		table.Rich(row, style)

	}
	table.Render()

	fmt.Println()
	term.PrintCmds("", "connect", "stop")

}
