package cmd

import (
	"fmt"
	"os"
	"plandex/lib"
	"strings"

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
	conversation, err := lib.LoadConversation()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error loading conversation:", err)
		return
	}

	if len(conversation) == 0 {
		fmt.Println("No conversation history.")
		return
	}

	var convMarkdown []string
	var totalTokens int
	for _, msg := range conversation {
		var author string
		if msg.Message.Role == "assistant" {
			author = "Plandex"
		} else if msg.Message.Role == "user" {
			author = "You"
		} else {
			author = msg.Message.Role
		}

		header := fmt.Sprintf("#### %s | %s | %d 🪙",
			author, msg.Timestamp, msg.Tokens)
		convMarkdown = append(convMarkdown, header, msg.Message.Content, "")
		totalTokens += msg.Tokens
	}

	markdownString, err := lib.GetMarkdown(strings.Join(convMarkdown, "\n"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error creating markdown representation:", err)
		return
	}

	fmt.Print(markdownString)

	fmt.Println(lib.GetDivisionLine())
	fmt.Println()
	fmt.Println(color.New(color.Bold, color.FgCyan).Sprint("  Conversation size →") + fmt.Sprintf(" %d 🪙", totalTokens))
	fmt.Println()
}
