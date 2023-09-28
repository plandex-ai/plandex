/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/spf13/cobra"
)

// contextRmCmd represents the delete command
var contextRmCmd = &cobra.Command{
	Use:     "rm",
	Aliases: []string{"delete"},
	Short:   "",
	Long:    ``,
	Run:     contextRm,
}

func contextRm(cmd *cobra.Command, args []string) {

}

func init() {
	contextCmd.AddCommand(contextRmCmd)

}
