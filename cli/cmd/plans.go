package cmd

import (
	"fmt"
	"os"
	"strconv"
	"time"

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
	Use:   "plans",
	Short: "List all available plans",
	Run:   plans,
}

func plans(cmd *cobra.Command, args []string) {
	plans, err := lib.GetPlans()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		return
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetAutoWrapText(false)
	table.SetHeader([]string{"#", "Name", "Updated", "Created", "Context", "Convo"})

	if len(plans) == 0 {
		fmt.Println("🤷‍♂️ No plans")
		fmt.Println()
		term.PrintCmds("", "new")
		return
	}

	for i, p := range plans {

		var name string
		if p.Name == lib.CurrentPlanName {
			name = color.New(color.Bold, color.FgGreen).Sprint(p.Name) + color.New(color.FgWhite).Sprint(" 👈 current")
		} else {
			name = p.Name
		}

		createdAt, err := time.Parse(shared.TsFormat, p.CreatedAt)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error parsing time:", err)
			continue
		}

		updatedAt, err := time.Parse(shared.TsFormat, p.UpdatedAt)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error parsing time:", err)
			continue
		}

		row := []string{
			strconv.Itoa(i + 1),
			name,
			format.Time(updatedAt),
			format.Time(createdAt),
			strconv.Itoa(p.ContextTokens) + " 🪙",
			strconv.Itoa(p.ConvoTokens) + " 🪙",
		}

		var style []tablewriter.Colors
		if p.Name == lib.CurrentPlanName {
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
	term.PrintCmds("", "new", "cd", "delete-plan")
}
