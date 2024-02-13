package cmd

import (
	"fmt"
	"os"
	"plandex/api"
	"plandex/auth"
	"plandex/format"
	"plandex/lib"
	"plandex/term"
	"strconv"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

var branchesCmd = &cobra.Command{
	Use:     "branches",
	Aliases: []string{"br"},
	Short:   "List plan branches",
	Run:     branches,
}

func init() {
	RootCmd.AddCommand(branchesCmd)
}

func branches(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()
	lib.MustResolveProject()

	if lib.CurrentPlanId == "" {
		fmt.Println("🤷‍♂️ No current plan")
		return
	}

	branches, apiErr := api.Client.ListBranches(lib.CurrentPlanId)

	if apiErr != nil {
		fmt.Println("Error getting branches:", apiErr)
		return
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetAutoWrapText(false)
	table.SetHeader([]string{"#", "Name", "Updated", "Created", "Context", "Convo"})

	for i, b := range branches {
		num := strconv.Itoa(i + 1)
		if b.Name == lib.CurrentBranch {
			num = color.New(color.Bold, color.FgGreen).Sprint(num)
		}

		var name string
		if b.Name == lib.CurrentBranch {
			name = color.New(color.Bold, color.FgGreen).Sprint(b.Name) + color.New(color.FgWhite).Sprint(" 👈 current")
		} else {
			name = b.Name
		}

		row := []string{
			num,
			name,
			format.Time(b.UpdatedAt),
			format.Time(b.CreatedAt),
			strconv.Itoa(b.ContextTokens) + " 🪙",
			strconv.Itoa(b.ConvoTokens) + " 🪙",
		}

		var style []tablewriter.Colors
		if b.Name == lib.CurrentPlanId {
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
	term.PrintCmds("", "checkout", "delete-branch")

}
