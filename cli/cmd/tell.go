package cmd

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"plandex/api"
	"plandex/auth"
	"plandex/lib"
	streamtui "plandex/stream_tui"
	"plandex/term"
	"strings"

	"github.com/plandex/plandex/shared"
	"github.com/spf13/cobra"
)

const defaultEditor = "vim"

var tellPromptFile string
var tellBg bool

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

	tellCmd.Flags().StringVarP(&tellPromptFile, "file", "f", "", "File containing prompt")

	tellCmd.Flags().BoolVar(&tellBg, "bg", false, "Execute autonomously in the background")
}

func tell(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()
	lib.MustResolveProject()

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

		if prompt != "" {
			fmt.Print("\n\n")
			fmt.Print(term.GetDivisionLine())
			fmt.Print("\n\n")
			fmt.Println(prompt)
			fmt.Print("\n\n")
			fmt.Print(term.GetDivisionLine())
			fmt.Print("\n\n")
		}
	}

	if prompt == "" {
		fmt.Fprintln(os.Stderr, "🤷‍♂️ No prompt to send")
		return
	}

	lib.MustCheckOutdatedContextWithOutput()

	if !tellBg {
		go func() {
			err := streamtui.StartStreamUI()

			if err != nil {
				fmt.Fprintln(os.Stderr, "Error starting stream UI:", err)
				os.Exit(1)
			}

			os.Exit(0)
		}()
	}

	apiErr := api.Client.TellPlan(lib.CurrentPlanId, shared.TellPlanRequest{
		Prompt:        prompt,
		ConnectStream: !tellBg,
	}, lib.OnStreamPlan)
	if apiErr != nil {
		fmt.Fprintln(os.Stderr, "Prompt error:", apiErr.Msg)
		return
	}

	if apiErr.Type == shared.ApiErrorTypeTrialMessagesExceeded {
		fmt.Fprintf(os.Stderr, "🚨 You've reached the free trial limit of %d messages per plan\n", apiErr.TrialMessagesExceededError.MaxMessages)

		res, err := term.ConfirmYesNo("Upgrade now?")

		if err != nil {
			fmt.Fprintln(os.Stderr, "Error prompting upgrade trial:", err)
			return
		}

		if res {
			err := auth.ConvertTrial()
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error converting trial:", err)
				return
			}
		}

		return
	}

	log.Println("Prompt sent")

	if tellBg {
		fmt.Println("✅ Prompt sent")
	} else {
		// Wait for stream UI to quit
		select {}
	}

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
