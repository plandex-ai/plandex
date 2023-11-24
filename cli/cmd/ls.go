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
	table.SetHeader([]string{"#", "Name", "Type", "🪙", "Added", "Updated"})
	table.SetAutoWrapText(false)

	if len(context) == 0 {
		fmt.Println("🤷‍♂️ No context")
		fmt.Println()
		term.PrintCmds("", "load")
		return
	}

	for i, part := range context {
		totalTokens += part.NumTokens

		t, icon := lib.GetContextTypeAndIcon(part)
		addedAt, err := time.Parse(shared.TsFormat, part.AddedAt)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error parsing time:", err)
			continue
		}

		updatedAt, err := time.Parse(shared.TsFormat, part.UpdatedAt)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error parsing time:", err)
			continue
		}

		row := []string{
			strconv.Itoa(i + 1),
			" " + icon + " " + part.Name,
			t,
			strconv.Itoa(part.NumTokens), //+ " 🪙",
			format.Time(addedAt),
			format.Time(updatedAt),
		}
		table.Rich(row, []tablewriter.Colors{
			{tablewriter.FgHiWhiteColor, tablewriter.Bold},
			{tablewriter.FgHiGreenColor, tablewriter.Bold},
			{tablewriter.FgHiWhiteColor},
			{tablewriter.FgHiWhiteColor},
			{tablewriter.FgHiWhiteColor},
			{tablewriter.FgHiWhiteColor},
		})
	}

	table.Render()

	tokensTbl := tablewriter.NewWriter(os.Stdout)
	tokensTbl.SetAutoWrapText(false)
	tokensTbl.Append([]string{color.New(color.FgHiCyan, color.Bold).Sprintf("Total tokens →") + color.New(color.FgHiWhite, color.Bold).Sprintf(" %d 🪙", totalTokens)})

	tokensTbl.Render()

	fmt.Println()
	term.PrintCmds("", "load", "rm", "clear")

}

func init() {
	RootCmd.AddCommand(contextCmd)

}
