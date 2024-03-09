package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version will be set at build time using -ldflags
var Version = "development"

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of Plandex",
	Long:  `All software has versions. This is Plandex's`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Plandex CLI Version:", Version)
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
