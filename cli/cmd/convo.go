package cmd

import (
	"fmt"
	"os"
	"plandex/lib"
	"plandex/term"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/plandex/plandex/shared"
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
	conversation, err := lib.LoadConversation()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error loading conversation:", err)
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
		if msg.Message.Role == "assistant" {
			author = "🤖 Plandex"
		} else if msg.Message.Role == "user" {
			author = "💬 You"
		} else {
			author = msg.Message.Role
		}

		dt, err := time.Parse(shared.TsFormat, msg.Timestamp)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error parsing time:", err)
			continue
		}

		// format as above but start with day of week
		formattedTs := dt.Local().Format("Mon Jan 2, 2006 | 3:04:05pm MST")

		// if it's today then use 'Today' instead of the date
		if dt.Day() == time.Now().Day() {
			formattedTs = dt.Local().Format("Today | 3:04:05pm MST")
		}

		// if it's yesterday then use 'Yesterday' instead of the date
		if dt.Day() == time.Now().AddDate(0, 0, -1).Day() {
			formattedTs = dt.Local().Format("Yesterday | 3:04:05pm MST")
		}

		header := fmt.Sprintf("#### %d | %s | %s | %d 🪙", i+1,
			author, formattedTs, msg.Tokens)
		convMarkdown = append(convMarkdown, header, msg.Message.Content, "")
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

	term.PageOutputReverse(output)
}
