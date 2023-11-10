package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"plandex/lib"
	"strconv"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/plandex/plandex/shared"
	"github.com/spf13/cobra"
)

var contextRmCmd = &cobra.Command{
	Use:     "rm",
	Aliases: []string{"remove"},
	Short:   "Remove context",
	Long:    `Remove context by index, name, or glob.`,
	Args:    cobra.MinimumNArgs(1),
	Run:     contextRm,
}

func contextRm(cmd *cobra.Command, args []string) {
	context, err := lib.GetAllContext(true)

	if err != nil {
		fmt.Fprintln(os.Stderr, "Error retrieving context:", err)
		return
	}

	toRemovePaths := []string{}
	toRemoveParts := []*shared.ModelContextPart{}

	for i, part := range context {
		path := lib.CreateContextFileName(part.Name, part.Sha)
		for _, id := range args {
			if fmt.Sprintf("%d", i+1) == id || part.Name == id || part.FilePath == id || part.Url == id {
				toRemovePaths = append(toRemovePaths, path)
				toRemoveParts = append(toRemoveParts, part)
				break
			} else if part.FilePath != "" {
				// Check if id is a glob pattern
				matched, err := filepath.Match(id, part.FilePath)
				if err != nil {
					fmt.Fprintln(os.Stderr, "Error matching glob pattern:", err)
					return
				}
				if matched {
					toRemovePaths = append(toRemovePaths, path)
					toRemoveParts = append(toRemoveParts, part)
					break
				}
			}
		}
	}

	err = lib.ContextRemoveFiles(toRemovePaths)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error removing context:", err)
		return
	}

	// update plan state with new token count
	planState, err := lib.GetPlanState()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error retrieving plan state:", err)
		return
	}

	tableString := &strings.Builder{}
	table := tablewriter.NewWriter(tableString)
	table.SetHeader([]string{"Name", "Type", "🪙"})
	table.SetAutoWrapText(false)

	removedTokens := 0
	totalTokens := planState.ContextTokens
	totalUpdatableTokens := planState.ContextUpdatableTokens
	for _, part := range toRemoveParts {
		removedTokens += part.NumTokens
		totalTokens -= part.NumTokens

		if part.Type == shared.ContextFileType || part.Type == shared.ContextDirectoryTreeType || part.Type == shared.ContextURLType {
			totalUpdatableTokens -= part.NumTokens
		}

		t, icon := lib.GetContextTypeAndIcon(part)
		row := []string{
			" " + icon + " " + part.Name,
			t,
			"-" + strconv.Itoa(part.NumTokens),
		}

		table.Rich(row, []tablewriter.Colors{
			{tablewriter.FgHiRedColor, tablewriter.Bold},
			{tablewriter.FgHiRedColor},
			{tablewriter.FgHiRedColor},
		})
	}
	planState.ContextTokens = totalTokens
	planState.ContextUpdatableTokens = totalUpdatableTokens

	err = lib.SetPlanState(planState, shared.StringTs())

	if err != nil {
		fmt.Fprintln(os.Stderr, "Error writing plan state:", err)
		return
	}

	// output
	if len(toRemovePaths) > 0 {
		suffix := ""
		if len(toRemovePaths) > 1 {
			suffix = "s"
		}
		msg := fmt.Sprintf("Removed %d piece%s of context | removed → %d 🪙 | total → %d 🪙 \n", len(toRemovePaths), suffix, removedTokens, totalTokens)
		table.Render()

		msg += "\n" + tableString.String()

		err = lib.GitCommitContextUpdate(msg)

		if err != nil {
			fmt.Fprintln(os.Stderr, "Error committing context update:", err)
			return
		}

		fmt.Println("✅ " + msg)

	} else {
		fmt.Println("🤷‍♂️ No context removed")
	}

}

func init() {
	RootCmd.AddCommand(contextRmCmd)
}
