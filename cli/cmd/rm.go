package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"plandex/lib"

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
			if fmt.Sprintf("%d", i) == id || part.Name == id || part.FilePath == id || part.Url == id {
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

	err = lib.ContextRm(toRemovePaths)
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

	removedTokens := 0
	totalTokens := planState.ContextTokens
	for _, part := range toRemoveParts {
		removedTokens += part.NumTokens
		totalTokens -= part.NumTokens
	}
	planState.ContextTokens = totalTokens

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
