package cmd

import (
	"fmt"
	"plandex-cli/auth"
	"plandex-cli/lib"
	"plandex-cli/term"

	"github.com/fatih/color"
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

var claudeStatusCmd = &cobra.Command{
	Use:   "claude-status",
	Short: "Check the status of your Claude Pro or Max subscription",
	Run:   claudeStatus,
}

func connectClaude(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()
	lib.ConnectClaudeMax()
}

func disconnectClaude(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()
	lib.DisconnectClaudeMax()
}

func claudeStatus(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()

	creds, err := lib.GetAccountCredentials()
	if err != nil {
		term.OutputErrorAndExit("Error getting account credentials: %v", err)
	}

	orgUserConfig := lib.MustGetOrgUserConfig()

	connected := creds.ClaudeMax != nil && orgUserConfig.UseClaudeSubscription

	if connected {
		fmt.Println("✅ Claude Pro or Max subscription is connected")

		// if orgUserConfig.IsClaudeSubscriptionCooldownActive() {
		if true {
			fmt.Println()
			color.New(term.ColorHiYellow, color.Bold).Println("⏳ You've reached your Claude Pro or Max subscription quota")
			fmt.Println("The next provider with valid credentials will be used for Anthropic models until the quota resets")
			fmt.Println()
		}

		term.PrintCmds("", "disconnect-claude")
	} else {
		fmt.Println("❌ No Claude Pro or Max subscription is connected")
		term.PrintCmds("", "connect-claude")
	}
}

func init() {
	RootCmd.AddCommand(connectClaudeCmd)
	RootCmd.AddCommand(disconnectClaudeCmd)
	RootCmd.AddCommand(claudeStatusCmd)
}
