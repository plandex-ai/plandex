package cmd

import "github.com/spf13/cobra"

func init() {
	RootCmd.AddCommand(modelsCmd)
}

var modelsCmd = &cobra.Command{
	Use:   "models",
	Short: "Show model settings",
	Run:   models,
}

func models(cmd *cobra.Command, args []string) {

}
