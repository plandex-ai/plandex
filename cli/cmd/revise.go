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

// reviseCmd represents the prompt command
var reviseCmd = &cobra.Command{
	Use:   "revise [prompt]",
	Short: "",
	Long:  ``,
	Run:   revise,
}

func revise(cmd *cobra.Command, args []string) {
	p := args[0]

	err := lib.Propose(p)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Prompt error:", err)
		return
	}
}

func init() {
	RootCmd.AddCommand(reviseCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// reviseCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// reviseCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
