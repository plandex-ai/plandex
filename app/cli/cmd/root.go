func init() {
	var helpCmd = &cobra.Command{
		Use:   "help",
		Short: "Display help for Plandex",
		Long:  `Display help for Plandex.`,
		Run: func(cmd *cobra.Command, args []string) {
			term.PrintCustomHelp()
		},
	}

	RootCmd.AddCommand(helpCmd)
}package cmd

import (
	"github.com/spf13/cobra"
	"plandex/term"
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use: `plandex [command] [flags]`,
	// Short: "Plandex: iterative development with AI",
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd, args)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		// term.OutputErrorAndExit("Error executing root command: %v", err)
		log.Fatalf("Error executing root command: %v", err)
	}
}

func run(cmd *cobra.Command, args []string) {

}
