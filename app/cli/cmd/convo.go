package cmd

import (
	"fmt"
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
	Short: "Display complete conversation history",
	Run:   convo,
}

func init() {
	RootCmd.AddCommand(convoCmd)
}

const stoppedEarlyMsg = "You stopped the reply early"

func convo(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()
	lib.MustResolveProject()

	term.StartSpinner("")
	conversation, apiErr := api.Client.ListConvo(lib.CurrentPlanId, lib.CurrentBranch)
	term.StopSpinner()

	if apiErr != nil {
		term.OutputErrorAndExit("Error loading conversation: %v", apiErr.Msg)
	}

	if len(conversation) == 0 {
		fmt.Println("🤷‍♂️ No conversation history")
		return
	}

	var convo string
	var totalTokens int
	for i, msg := range conversation {
		var author string
		if msg.Role == "assistant" {
			author = "🤖 Plandex"
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

		// convMarkdown = append(convMarkdown, header, msg.Message, "")

		md, err := term.GetMarkdown(header + "\n" + msg.Message + "\n\n")
		if err != nil {
			term.OutputErrorAndExit("Error creating markdown representation: %v", err)
		}
		convo += md

		if msg.Stopped {
			convo += fmt.Sprintf(" 🛑 %s\n\n", color.New(color.Bold).Sprint(stoppedEarlyMsg))
		}

		totalTokens += msg.Tokens
	}

	convo = strings.ReplaceAll(convo, stoppedEarlyMsg, color.New(term.ColorHiRed).Sprint(stoppedEarlyMsg))

	output :=
		fmt.Sprintf("\n%s", convo) +
			term.GetDivisionLine() +
			color.New(color.Bold, term.ColorHiCyan).Sprint("  Conversation size →") + fmt.Sprintf(" %d 🪙", totalTokens) + "\n\n"

	term.PageOutput(output)
}
