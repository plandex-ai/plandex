/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"
	"plandex/lib"

	"github.com/spf13/cobra"
)

// contextLsCmd represents the ls command
var contextLsCmd = &cobra.Command{
	Use:     "ls",
	Aliases: []string{"list"},
	Short:   "",
	Long:    ``,
	Run:     contextLs,
}

func contextLs(cmd *cobra.Command, args []string) {
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
	contextCmd.AddCommand(contextLsCmd)

}
