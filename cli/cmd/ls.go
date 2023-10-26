package cmd

import (
	"fmt"
	"os"
	"plandex/lib"

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
		fmt.Fprintln(os.Stderr, "Error listing context:", err)
		os.Exit(1)
	}

	totalTokens := 0

	for i, part := range context {
		totalTokens += part.NumTokens

		if i != 0 {
			fmt.Print("\n")
		}

		fmt.Println("Index:", i)
		if part.FilePath != "" {
			fmt.Printf("File: %s\n", part.FilePath)
		}

		if part.Url != "" {
			fmt.Printf("Url: %s\n", part.Url)
		}

		fmt.Printf("Tokens: %d\n", part.NumTokens)
		fmt.Printf("Updated: %s\n", part.UpdatedAt)
	}

	fmt.Printf("\nTotal tokens: %d\n", totalTokens)

}

func init() {
	RootCmd.AddCommand(contextCmd)

}
