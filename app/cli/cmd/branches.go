package cmd

import (
	"fmt"
	"os"
	"plandex-cli/api"
	"plandex-cli/auth"
	"plandex-cli/format"
	"plandex-cli/lib"
	"plandex-cli/term"
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
		term.OutputNoCurrentPlanErrorAndExit()
	}

	term.StartSpinner("")

	branches, apiErr := api.Client.ListBranches(lib.CurrentPlanId)

	term.StopSpinner()

	if apiErr != nil {
		term.OutputErrorAndExit("Error getting branches: %v", apiErr)
		return
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetAutoWrapText(false)
	table.SetHeader([]string{"#", "Name", "Updated" /* "Created",*/, "Context", "Convo"})

	for i, b := range branches {
		num := strconv.Itoa(i + 1)
		if b.Name == lib.CurrentBranch {
			num = color.New(color.Bold, term.ColorHiGreen).Sprint(num)
		}

		var name string
		if b.Name == lib.CurrentBranch {
			name = color.New(color.Bold, term.ColorHiGreen).Sprint(b.Name) + " 👈"
		} else {
			name = b.Name
		}

		row := []string{
			num,
			name,
			format.Time(b.UpdatedAt),
			// format.Time(b.CreatedAt),
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
				{tablewriter.Bold},
			}
		}

		table.Rich(row, style)

	}
	table.Render()
	fmt.Println()
	term.PrintCmds("", "checkout", "delete-branch")

}
