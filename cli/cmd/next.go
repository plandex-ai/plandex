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

// Variables to be used in the nextCmd
var prompt string = "continue to the next step of the plan"

// nextCmd represents the prompt command
var nextCmd = &cobra.Command{
	Use:     "next",
	Aliases: []string{"n"},
	Short:   "Continue to the next step of the plan.",
	Run:     next,
}

func init() {
	RootCmd.AddCommand(nextCmd)
}

func next(cmd *cobra.Command, args []string) {
	err := lib.Propose(prompt)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Prompt error:", err)
		return
	}
}