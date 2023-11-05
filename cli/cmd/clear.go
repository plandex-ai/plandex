package cmd

import (
	"fmt"
	"os"
	"plandex/lib"

	"github.com/plandex/plandex/shared"
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
	context, err := lib.GetAllContext(true)

	if err != nil {
		fmt.Fprintln(os.Stderr, "Error retrieving context:", err)
		return
	}

	toRemovePaths := []string{}
	for _, part := range context {
		path := lib.CreateContextFileName(part.Name, part.Sha)
		toRemovePaths = append(toRemovePaths, path)
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
	planState.ContextTokens = 0

	err = lib.SetPlanState(planState, shared.StringTs())

	if err != nil {
		fmt.Fprintln(os.Stderr, "Error writing plan state:", err)
		return
	}

	msg := "Context cleared"
	err = lib.GitCommitContextUpdate(msg)

	if err != nil {
		fmt.Fprintln(os.Stderr, "Error committing context update:", err)
		return
	}

	fmt.Println("✅ " + msg)

}

func init() {
	RootCmd.AddCommand(clearCmd)
}
