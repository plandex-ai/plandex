package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"plandex/lib"
	"plandex/types"

	"github.com/spf13/cobra"
)

var clearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear all context",
	Long:  `Clear all context.`,
	Run:   clearAllContext,
}

func clearAllContext(cmd *cobra.Command, args []string) {
	// clear all files from context dir
	err := os.RemoveAll(lib.ContextSubdir)

	if err != nil {
		fmt.Fprintln(os.Stderr, "Error clearing context:", err)
		return
	}

	// create context dir
	err = os.MkdirAll(lib.ContextSubdir, 0755)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error creating context dir:", err)
		return
	}

	contextStateFilePath := filepath.Join(lib.ContextSubdir, "context.json")
	contextState := types.ModelContextState{NumTokens: 0}
	bytes, err := json.MarshalIndent(contextState, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error marshalling context state:", err)
		return
	}

	err = os.WriteFile(contextStateFilePath, bytes, 0644)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error writing context state file:", err)
		return
	}

	fmt.Println("✅ All context cleared")

}

func init() {
	RootCmd.AddCommand(clearCmd)
}
