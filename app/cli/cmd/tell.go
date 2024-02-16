package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"plandex/auth"
	"plandex/lib"
	"plandex/tell"
	"plandex/term"
	"strings"

	"github.com/spf13/cobra"
)

const defaultEditor = "vim"

// const defaultEditor = "nano"

var tellPromptFile string
var tellBg bool
var tellStop bool
var tellNoBuild bool

// tellCmd represents the prompt command
var tellCmd = &cobra.Command{
	Use:     "tell [prompt]",
	Aliases: []string{"t"},
	Short:   "Send a prompt for the current plan.",
	// Long:  ``,
	Args: cobra.RangeArgs(0, 1),
	Run:  doTell,
}

func init() {
	RootCmd.AddCommand(tellCmd)

	tellCmd.Flags().StringVarP(&tellPromptFile, "file", "f", "", "File containing prompt")
	// tellCmd.Flags().BoolVar(&tellBg, "bg", false, "Execute autonomously in the background") // Not implemented yet
	tellCmd.Flags().BoolVarP(&tellStop, "stop", "s", false, "Stop after a single reply")
	tellCmd.Flags().BoolVarP(&tellNoBuild, "no-build", "n", false, "Don't build files")
}

func doTell(cmd *cobra.Command, args []string) {
	if os.Getenv("OPENAI_API_KEY") == "" {
		term.OutputNoApiKeyMsg()
		os.Exit(1)
	}

	auth.MustResolveAuthWithOrg()
	lib.MustResolveProject()

	if lib.CurrentPlanId == "" {
		fmt.Fprintln(os.Stderr, "No current plan")
		return
	}

	var prompt string

	if len(args) > 0 {
		prompt = args[0]
	} else if tellPromptFile != "" {
		bytes, err := os.ReadFile(tellPromptFile)
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

		instructions := getEditorInstructions(editor)
		filename := tempFile.Name()
		err = os.WriteFile(filename, []byte(instructions), 0644)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Failed to write instructions to temporary file:", err)
			return
		}

		editorCmd := prepareEditorCommand(editor, filename)
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

		prompt = strings.TrimPrefix(prompt, strings.TrimSpace(instructions))
		prompt = strings.TrimSpace(prompt)
	}

	if prompt == "" {
		fmt.Fprintln(os.Stderr, "🤷‍♂️ No prompt to send")
		return
	}

	lib.MustCheckOutdatedContextWithOutput()

	tell.TellPlan(prompt, tellBg, tellStop, tellNoBuild, false)
}

func prepareEditorCommand(editor string, filename string) *exec.Cmd {
	switch editor {
	case "vim":
		return exec.Command(editor, "+normal G$", "+startinsert!", filename)
	case "nano":
		return exec.Command(editor, "+99999999", filename)
	default:
		return exec.Command(editor, filename)
	}
}

func getEditorInstructions(editor string) string {

	return "Write your prompt below, then save and exit to send it to Plandex.\n\n"

}
