package cmd

import (
	"fmt"
	"os"
	"plandex/lib"
	"strconv"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

var contextCmd = &cobra.Command{
	Use:     "ls",
	Aliases: []string{"ls"},
	Short:   "List everything in context",
	Run:     context,
}

func context(cmd *cobra.Command, args []string) {
	context, err := lib.GetAllContext(true)

	if err != nil {
		color.New(color.FgRed).Fprintln(os.Stderr, "Error listing context:", err)
		os.Exit(1)
	}

	totalTokens := 0
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Index", "Type", "ID", "Tokens", "Updated"})
	table.SetAutoWrapText(false)

	for i, part := range context {
		totalTokens += part.NumTokens

		var contextType string
		var id string
		if part.FilePath != "" {
			contextType = "file"
			id = part.FilePath
		} else if part.Url != "" {
			contextType = "url"
			id = part.Url
		} else {
			contextType = "text"
			id = part.Name
		}

		row := []string{
			strconv.Itoa(i),
			contextType,
			id,
			strconv.Itoa(part.NumTokens) + " 🪙",
			part.UpdatedAt,
		}
		if i%2 == 0 {
			table.Rich(row, []tablewriter.Colors{
				{tablewriter.FgHiCyanColor, tablewriter.Bold},
				{tablewriter.FgHiWhiteColor},
				{tablewriter.FgHiGreenColor, tablewriter.Bold},
				{tablewriter.FgHiWhiteColor},
				{tablewriter.FgHiWhiteColor},
			})
		} else {
			table.Append(row)
		}
	}

	table.Render()
	fmt.Print(color.New(color.FgHiCyan, color.Bold).Sprintf("\nTotal →") + color.New(color.FgHiWhite, color.Bold).Sprintf(" %d 🪙\n", totalTokens))

}

func init() {
	RootCmd.AddCommand(contextCmd)

}
