package cmd

import (
	"plandex/term"

	"github.com/spf13/cobra"
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   `plandex [command] [flags]`,
	Short: "Plandex: iterative development with AI",
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd, args)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		term.OutputErrorAndExit("Error executing root command: %v", err)
	}
}

func run(cmd *cobra.Command, args []string) {

}
