package cmd

import (
	"plandex-cli/lib"

	"github.com/spf13/cobra"
)

var connectClaudeCmd = &cobra.Command{
	Use:   "connect-claude",
	Short: "Connect your Claude Pro or Max subscription",
	Run:   connectClaude,
}

var disconnectClaudeCmd = &cobra.Command{
	Use:   "disconnect-claude",
	Short: "Disconnect your Claude Pro or Max subscription",
	Run:   disconnectClaude,
}

func connectClaude(cmd *cobra.Command, args []string) {
	lib.ConnectClaudeMax()
}

func disconnectClaude(cmd *cobra.Command, args []string) {
	lib.DisconnectClaudeMax()
}

func init() {
	RootCmd.AddCommand(connectClaudeCmd)
	RootCmd.AddCommand(disconnectClaudeCmd)
}
