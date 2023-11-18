package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"plandex/lib"
	"strings"

	"github.com/spf13/cobra"
)

const defaultEditor = "vim"

var promptFile string

// tellCmd represents the prompt command
var tellCmd = &cobra.Command{
	Use:     "tell [prompt]",
	Aliases: []string{"t"},
	Short:   "Send a prompt for the current plan.",
	// Long:  ``,
	Args: cobra.RangeArgs(0, 1),
	Run:  tell,
}

func init() {
	RootCmd.AddCommand(tellCmd)

	tellCmd.Flags().StringVarP(&promptFile, "file", "f", "", "File containing prompt")
}

func tell(cmd *cobra.Command, args []string) {
	var prompt string

	if len(args) > 0 {
		prompt = args[0]
	} else if promptFile != "" {
		bytes, err := os.ReadFile(promptFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error reading prompt file:", err)
			return
		}
		prompt = string(bytes)
	} else {
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = os.Getenv("VISUAL")
			if editor == "" {
				editor = defaultEditor
			}
		}

		tempFile, err := os.CreateTemp(os.TempDir(), "plandex_prompt_*")
		if err != nil {
			fmt.Fprintln(os.Stderr, "Failed to create temporary file:", err)
			return
		}

		instruction := "Write your prompt below, then save and exit to send it to Plandex.\n\n"
		_, err = tempFile.WriteString(instruction)

		editorCmd := prepareEditorCommand(editor, tempFile.Name())
		editorCmd.Stdin = os.Stdin
		editorCmd.Stdout = os.Stdout
		editorCmd.Stderr = os.Stderr
		err = editorCmd.Run()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error opening editor:", err)
			return
		}

		bytes, err := os.ReadFile(tempFile.Name())
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error reading temporary file:", err)
			return
		}

		prompt = string(bytes)

		err = os.Remove(tempFile.Name())
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error removing temporary file:", err)
			return
		}

		prompt = strings.TrimPrefix(prompt, strings.TrimSpace(instruction))
		prompt = strings.TrimSpace(prompt)

		if prompt != "" {
			fmt.Print("Prompt:\n\n")
			fmt.Println(prompt)
		}
	}

	if prompt == "" {
		fmt.Fprintln(os.Stderr, "🤷‍♂️ No prompt to send.")
		return
	}

	// fmt.Println("lib.Propose(prompt)")

	err := lib.Propose(prompt)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Prompt error:", err)
		return
	}

	// fmt.Println("lib.Propose(prompt) success")
}

func prepareEditorCommand(editor string, filename string) *exec.Cmd {
	switch editor {
	case "vim":
		vimInstructions := "Write your prompt below, then save and exit to send it to Plandex.\n\n" +
			"Vim Instructions:\n" +
			"Use the arrow keys or 'hjkl' to navigate around the text.\n" +
			"Press 'i' to enter 'Insert' mode and start typing your prompt.\n" +
			"To exit 'Insert' mode and go back to 'Normal' mode, press 'Esc'.\n" +
			"In 'Normal' mode, type ':wq' followed by 'Enter' key to save and exit.\n" +
			"If you want to exit without saving, type ':q!' followed by 'Enter'."

		// Write the Vim specific instructions into the temporary file
		err := os.WriteFile(filename, []byte(vimInstructions), 0644)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Failed to write instructions to temporary file:", err)
			return nil
		}

		return exec.Command(editor, filename)

	case "nano":
		nanoInstructions := "Write your prompt below, then save and exit to send it to Plandex.\n\n" +
			"Nano Instructions:\n" +
			"Use the arrow keys to navigate around the text.\n" +
			"Type your prompt as needed.\n" +
			"When you're ready, press 'Ctrl + O' to save the file, then 'Enter' to confirm.\n" +
			"Finally, press 'Ctrl + X' to exit Nano and send your prompt."

		// Write the Nano specific instructions into the temporary file
		err := os.WriteFile(filename, []byte(nanoInstructions), 0644)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Failed to write instructions to temporary file:", err)
			return nil
		}

		return exec.Command(editor, filename)
	default:
		return exec.Command(editor, filename)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to write instructions to temporary file:", err)
		return nil
	}

	switch editor {
	case "vim":
		return exec.Command(editor, "+normal G$", "+startinsert!", filename)
	case "nano":
		return exec.Command(editor, "+99999999", filename)
	default:
		return exec.Command(editor, filename)
	}
}
