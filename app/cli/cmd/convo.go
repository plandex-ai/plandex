package cmd

import (
	"fmt"
	"os"
	"plandex/api"
	"plandex/auth"
	"plandex/lib"
	"plandex/term"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// convoCmd represents the convo command
var convoCmd = &cobra.Command{
	Use:   "convo",
	Short: "Display complete conversation history.",
	Run:   convo,
}

func init() {
	RootCmd.AddCommand(convoCmd)
}

func convo(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()
	lib.MustResolveProject()

	conversation, apiErr := api.Client.ListConvo(lib.CurrentPlanId, lib.CurrentBranch)
	if apiErr != nil {
		fmt.Fprintln(os.Stderr, "Error loading conversation:", apiErr.Msg)
		return
	}

	if len(conversation) == 0 {
		fmt.Println("🤷‍♂️ No conversation history")
		return
	}

	var convMarkdown []string
	var totalTokens int
	for i, msg := range conversation {
		var author string
		if msg.Role == "assistant" {
			author = "🤖 Plandex"
			if msg.Stopped {
				author += " | 🛑 stopped early"
			}
		} else if msg.Role == "user" {
			author = "💬 You"
		} else {
			author = msg.Role
		}

		// format as above but start with day of week
		formattedTs := msg.CreatedAt.Local().Format("Mon Jan 2, 2006 | 3:04pm MST")

		// if it's today then use 'Today' instead of the date
		if msg.CreatedAt.Day() == time.Now().Day() {
			formattedTs = msg.CreatedAt.Local().Format("Today | 3:04pm MST")
		}

		// if it's yesterday then use 'Yesterday' instead of the date
		if msg.CreatedAt.Day() == time.Now().AddDate(0, 0, -1).Day() {
			formattedTs = msg.CreatedAt.Local().Format("Yesterday | 3:04pm MST")
		}

		header := fmt.Sprintf("#### %d | %s | %s | %d 🪙 ", i+1,
			author, formattedTs, msg.Tokens)
		convMarkdown = append(convMarkdown, header, msg.Message, "")
		totalTokens += msg.Tokens
	}

	markdownString, err := term.GetMarkdown(strings.Join(convMarkdown, "\n"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error creating markdown representation:", err)
		return
	}

	output :=
		fmt.Sprintf("\n%s", markdownString) +
			term.GetDivisionLine() +
			color.New(color.Bold, color.FgCyan).Sprint("  Conversation size →") + fmt.Sprintf(" %d 🪙", totalTokens) + "\n\n"

	term.PageOutput(output)
}
