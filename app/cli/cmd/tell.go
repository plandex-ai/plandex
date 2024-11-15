package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"plandex/auth"
	"plandex/lib"
	"plandex/plan_exec"
	"plandex/term"
	"strings"

	"github.com/plandex/plandex/shared"
	"github.com/spf13/cobra"
)

const defaultEditor = "vim"

// const defaultEditor = "nano"

var tellPromptFile string
var tellBg bool
var tellStop bool
var tellNoBuild bool
var tellAutoApply bool

// tellCmd represents the prompt command
var tellCmd = &cobra.Command{
	Use:     "tell [prompt]",
	Aliases: []string{"t"},
	Short:   "Send a prompt for the current plan",
	// Long:  ``,
	Args: cobra.RangeArgs(0, 1),
	Run:  doTell,
}

func init() {
	RootCmd.AddCommand(tellCmd)

	tellCmd.Flags().StringVarP(&tellPromptFile, "file", "f", "", "File containing prompt")
	tellCmd.Flags().BoolVarP(&tellStop, "stop", "s", false, "Stop after a single reply")
	tellCmd.Flags().BoolVarP(&tellNoBuild, "no-build", "n", false, "Don't build files")
	tellCmd.Flags().BoolVar(&tellBg, "bg", false, "Execute autonomously in the background")

	tellCmd.Flags().BoolVarP(&autoConfirm, "yes", "y", false, "Automatically confirm context updates")
	tellCmd.Flags().BoolVar(&tellAutoApply, "apply", false, "Automatically apply changes (and confirm context updates)")
	tellCmd.Flags().BoolVarP(&autoCommit, "commit", "c", false, "Commit changes to git when --apply/-a is passed")

}

func doTell(cmd *cobra.Command, args []string) {
	validateTellFlags()

	auth.MustResolveAuthWithOrg()
	lib.MustResolveProject()

	if lib.CurrentPlanId == "" {
		term.OutputNoCurrentPlanErrorAndExit()
	}

	var apiKeys map[string]string
	if !auth.Current.IntegratedModelsMode {
		apiKeys = lib.MustVerifyApiKeys()
	}

	prompt := getTellPrompt(args)

	if prompt == "" {
		fmt.Println("🤷‍♂️ No prompt to send")
		return
	}

	plan_exec.TellPlan(plan_exec.ExecParams{
		CurrentPlanId: lib.CurrentPlanId,
		CurrentBranch: lib.CurrentBranch,
		ApiKeys:       apiKeys,
		CheckOutdatedContext: func(maybeContexts []*shared.Context) (bool, bool, error) {
			return lib.CheckOutdatedContextWithOutput(false, autoConfirm || tellAutoApply, maybeContexts)
		},
	}, prompt, tellBg, tellStop, tellNoBuild, false, false, false)

	if tellAutoApply {
		lib.MustApplyPlan(lib.CurrentPlanId, lib.CurrentBranch, true, autoCommit, !autoCommit)
	}
}

func getTellPrompt(args []string) string {
	var prompt string
	var pipedData string

	if len(args) > 0 {
		prompt = args[0]
	} else if tellPromptFile != "" {
		bytes, err := os.ReadFile(tellPromptFile)
		if err != nil {
			term.OutputErrorAndExit("Error reading prompt file: %v", err)
		}
		prompt = string(bytes)
	}

	// Check if there's piped input
	fileInfo, err := os.Stdin.Stat()
	if err != nil {
		term.OutputErrorAndExit("Failed to stat stdin: %v", err)
	}

	if fileInfo.Mode()&os.ModeNamedPipe != 0 {
		reader := bufio.NewReader(os.Stdin)
		pipedDataBytes, err := io.ReadAll(reader)
		if err != nil {
			term.OutputErrorAndExit("Failed to read piped data: %v", err)
		}
		pipedData = string(pipedDataBytes)
	}

	if prompt == "" && pipedData == "" {
		prompt = getEditorPrompt()
	} else if pipedData != "" {
		if prompt != "" {
			prompt = fmt.Sprintf("%s\n\n---\n\n%s", prompt, pipedData)
		} else {
			prompt = pipedData
		}
	}

	return prompt
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
	return "👉  Write your prompt below, then save and exit to send it to Plandex.\n• To save and exit, press ESC, then type :wq! and press ENTER.\n• To exit without saving, press ESC, then type :q! and press ENTER.\n\n\n"
}

func getEditorPrompt() string {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
		if editor == "" {
			editor = defaultEditor
		}
	}

	tempFile, err := os.CreateTemp(os.TempDir(), "plandex_prompt_*")
	if err != nil {
		term.OutputErrorAndExit("Failed to create temporary file: %v", err)
	}

	instructions := getEditorInstructions(editor)
	filename := tempFile.Name()
	err = os.WriteFile(filename, []byte(instructions), 0644)
	if err != nil {
		term.OutputErrorAndExit("Failed to write instructions to temporary file: %v", err)
	}

	editorCmd := prepareEditorCommand(editor, filename)
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr
	err = editorCmd.Run()
	if err != nil {
		term.OutputErrorAndExit("Error opening editor: %v", err)
	}

	bytes, err := os.ReadFile(tempFile.Name())
	if err != nil {
		term.OutputErrorAndExit("Error reading temporary file: %v", err)
	}

	prompt := string(bytes)

	err = os.Remove(tempFile.Name())
	if err != nil {
		term.OutputErrorAndExit("Error removing temporary file: %v", err)
	}

	prompt = strings.TrimPrefix(prompt, strings.TrimSpace(instructions))
	prompt = strings.TrimSpace(prompt)

	return prompt

}

func validateTellFlags() {
	if tellAutoApply && tellNoBuild {
		term.OutputErrorAndExit("🚨 --apply/-a can't be used with --no-build/-n")
	}
	if tellAutoApply && tellBg {
		term.OutputErrorAndExit("🚨 --apply/-a can't be used with --bg")
	}
	if autoCommit && !tellAutoApply {
		term.OutputErrorAndExit("🚨 --commit/-c can only be used with --apply/-a")
	}
}
