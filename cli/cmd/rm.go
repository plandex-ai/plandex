package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"plandex/lib"
	"plandex/types"

	"github.com/plandex/plandex/shared"
	"github.com/spf13/cobra"
)

var contextRmCmd = &cobra.Command{
	Use:     "rm",
	Aliases: []string{"remove"},
	Short:   "Remove a specific context file",
	Long:    `This command allows the user to remove a specific context file by providing the file's index as an argument.`,
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
	toRemoveParts := []shared.ModelContextPart{}

	for i, part := range context {
		path := lib.CreateContextFileName(part.Name, part.Sha)
		for _, id := range args {
			if fmt.Sprintf("%d", i) == id || part.FilePath == id || part.Url == id {
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

	// remove files
	errCh := make(chan error, len(toRemovePaths)*2)
	for _, path := range toRemovePaths {
		for _, ext := range []string{".meta", ".body"} {
			go func(path, ext string) {
				errCh <- os.Remove(filepath.Join(lib.ContextSubdir, path+ext))
			}(path, ext)
		}
	}

	for i := 0; i < len(toRemovePaths)*2; i++ {
		err := <-errCh
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error removing context file:", err)
			return
		}
	}

	// update context.json with new token count
	var contextState types.ModelContextState

	contextStateFilePath := filepath.Join(lib.ContextSubdir, "context.json")
	bytes, err := os.ReadFile(contextStateFilePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error reading context state file:", err)
		return
	}

	err = json.Unmarshal(bytes, &contextState)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error unmarshalling context state:", err)
		return
	}

	removedTokens := 0
	totalTokens := contextState.NumTokens
	for _, part := range toRemoveParts {
		removedTokens += part.NumTokens
		totalTokens -= part.NumTokens
	}
	contextState.NumTokens = totalTokens

	bytes, err = json.MarshalIndent(contextState, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error marshalling context state:", err)
		return
	}

	err = os.WriteFile(contextStateFilePath, bytes, 0644)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error writing context state file:", err)
		return
	}

	// output
	if len(toRemovePaths) > 0 {
		suffix := ""
		if len(toRemovePaths) > 1 {
			suffix = "s"
		}
		fmt.Printf("✅ Removed %d piece%s of context | removed → %d 🪙 | total → %d 🪙 \n", len(toRemovePaths), suffix, removedTokens, totalTokens)
	} else {
		fmt.Println("🤷‍♂️ No context removed")
	}

}

func init() {
	RootCmd.AddCommand(contextRmCmd)
}
