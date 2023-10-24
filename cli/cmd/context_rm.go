/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"log"
	"os"
	"path/filepath"
	"plandex/lib"

	"github.com/spf13/cobra"
)

// contextRmCmd represents the delete command
var contextRmCmd = &cobra.Command{
	Use:   "rm [indices-or-files-or-urls...]",
	Short: "Remove context by index, filename, or url.",
	Args:  cobra.MinimumNArgs(1),
	Run:   contextRm,
}

func contextRm(cmd *cobra.Command, args []string) {

	contextStateFilePath := filepath.Join(lib.ContextSubdir, "context.json")

	contextStateFile, err := os.Open(contextStateFilePath)
	if err != nil {
		log.Fatalf("Error opening file: %v", err)
	}
	defer contextStateFile.Close()

}

func init() {
	contextCmd.AddCommand(contextRmCmd)

}
