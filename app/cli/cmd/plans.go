package cmd

import (
	"fmt"
	"os"
	"strconv"

	"plandex/api"
	"plandex/auth"
	"plandex/format"
	"plandex/lib"
	"plandex/term"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/plandex/plandex/shared"
	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(plansCmd)
}

// plansCmd represents the list command
var plansCmd = &cobra.Command{
	Use:     "plans",
	Aliases: []string{"pl"},
	Short:   "List all available plans",
	Run:     plans,
}

func plans(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()
	lib.MustResolveProject()

	plans, apiErr := api.Client.ListPlans([]string{lib.CurrentProjectId})

	if apiErr != nil {
		fmt.Fprintln(os.Stderr, "Error getting plans:", apiErr)
		return
	}

	if len(plans) == 0 {
		fmt.Println("🤷‍♂️ No plans")
		fmt.Println()
		term.PrintCmds("", "new")
		return
	}

	var planIds []string
	for _, p := range plans {
		planIds = append(planIds, p.Id)
	}

	currentBranchNamesByPlanId, err := lib.GetCurrentBranchNamesByPlanId(planIds)

	if err != nil {
		fmt.Fprintln(os.Stderr, "Error getting current branches:", err)
		return
	}

	currentBranchesByPlanId, apiErr := api.Client.GetCurrentBranchByPlanId(lib.CurrentProjectId, shared.GetCurrentBranchByPlanIdRequest{
		CurrentBranchByPlanId: currentBranchNamesByPlanId,
	})

	if apiErr != nil {
		fmt.Fprintln(os.Stderr, "Error getting current branches:", apiErr)
		return
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetAutoWrapText(false)
	table.SetHeader([]string{"#", "Name", "Updated", "Created" /*"Branches",*/, "Branch", "Context", "Convo"})

	for i, p := range plans {
		num := strconv.Itoa(i + 1)
		if p.Id == lib.CurrentPlanId {
			num = color.New(color.Bold, color.FgGreen).Sprint(num)
		}

		var name string
		if p.Id == lib.CurrentPlanId {
			name = color.New(color.Bold, color.FgGreen).Sprint(p.Name) + color.New(color.FgWhite).Sprint(" 👈 current")
		} else {
			name = p.Name
		}

		currentBranch := currentBranchesByPlanId[p.Id]

		row := []string{
			num,
			name,
			format.Time(p.UpdatedAt),
			format.Time(p.CreatedAt),
			// strconv.Itoa(p.ActiveBranches),
			currentBranch.Name,
			strconv.Itoa(currentBranch.ContextTokens) + " 🪙",
			strconv.Itoa(currentBranch.ConvoTokens) + " 🪙",
		}

		var style []tablewriter.Colors
		if p.Name == lib.CurrentPlanId {
			style = []tablewriter.Colors{
				{tablewriter.FgGreenColor, tablewriter.Bold},
			}
		} else {
			style = []tablewriter.Colors{
				{tablewriter.FgHiWhiteColor, tablewriter.Bold},
				{tablewriter.FgHiWhiteColor},
			}
		}

		table.Rich(row, style)

	}
	table.Render()
	fmt.Println()

	term.PrintCmds("", "tell", "new", "cd", "delete-plan")
}
