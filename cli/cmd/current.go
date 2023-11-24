package cmd

import (
	"fmt"
	"os"
	"plandex/format"
	"plandex/lib"
	"plandex/term"
	"strconv"
	"time"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/plandex/plandex/shared"
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
	if lib.CurrentPlanName == "" {
		fmt.Println("🤷‍♂️ No current plan")
		return
	}

	planState, err := lib.GetPlanState()
	if err != nil {
		fmt.Println("Error getting plan state:", err)
		return
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetAutoWrapText(false)
	table.SetHeader([]string{"Current Plan", "Updated", "Created", "Context", "Convo"})

	var name string
	if planState.Name == lib.CurrentPlanName {
		name = color.New(color.Bold, color.FgGreen).Sprint(planState.Name)
	} else {
		name = planState.Name
	}

	createdAt, err := time.Parse(shared.TsFormat, planState.CreatedAt)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error parsing time:", err)
		return
	}

	updatedAt, err := time.Parse(shared.TsFormat, planState.UpdatedAt)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error parsing time:", err)
		return
	}

	row := []string{
		name,
		format.Time(updatedAt),
		format.Time(createdAt),
		strconv.Itoa(planState.ContextTokens) + " 🪙",
		strconv.Itoa(planState.ConvoTokens) + " 🪙",
	}

	var style []tablewriter.Colors
	if planState.Name == lib.CurrentPlanName {
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

	table.Render()
	fmt.Println()
	term.PrintCmds("", "tell", "ls", "plans")

}
