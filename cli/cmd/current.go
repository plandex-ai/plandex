package cmd

import (
	"fmt"
	"os"
	"plandex/api"
	"plandex/format"
	"plandex/lib"
	"plandex/term"
	"strconv"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

var currentCmd = &cobra.Command{
	Use:     "current",
	Aliases: []string{"cu"},
	Short:   "Get the current plan",
	Run:     current,
}

func init() {
	RootCmd.AddCommand(currentCmd)
}

func current(cmd *cobra.Command, args []string) {
	lib.MustResolveProject()

	if lib.CurrentPlanId == "" {
		fmt.Println("🤷‍♂️ No current plan")
		return
	}

	plan, err := api.Client.GetPlan(lib.CurrentPlanId)
	if err != nil {
		fmt.Println("Error getting plan:", err)
		return
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetAutoWrapText(false)
	table.SetHeader([]string{"Current Plan", "Updated", "Created", "Context", "Convo"})

	name := color.New(color.Bold, color.FgGreen).Sprint(plan.Name)

	row := []string{
		name,
		format.Time(plan.UpdatedAt),
		format.Time(plan.CreatedAt),
		strconv.Itoa(plan.ContextTokens) + " 🪙",
		strconv.Itoa(plan.ConvoTokens) + " 🪙",
	}

	style := []tablewriter.Colors{
		{tablewriter.FgGreenColor, tablewriter.Bold},
	}

	table.Rich(row, style)

	table.Render()
	fmt.Println()
	term.PrintCmds("", "tell", "ls", "plans")

}
