package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"plandex/api"
	"plandex/auth"
	"plandex/lib"
	"plandex/plan_exec"
	"plandex/term"
	"strconv"
	"strings"
	"time"

	"context"

	"os/signal"
	"syscall"

	"github.com/plandex/plandex/shared"
	"github.com/spf13/cobra"
)

const defaultEditor = "vim"
const defaultAutoDebugTries = 5

var autoConfirm bool

// const defaultEditor = "nano"

var tellPromptFile string
var tellBg bool
var tellStop bool
var tellNoBuild bool
var tellAutoApply bool
var tellAutoContext bool
var noExec bool
var autoDebug int

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
	tellCmd.Flags().BoolVarP(&autoCommit, "commit", "c", false, "Commit changes to git when --apply is passed")

	tellCmd.Flags().BoolVar(&tellAutoContext, "auto-context", false, "Load and manage context automatically")

	tellCmd.Flags().BoolVar(&noExec, "no-exec", false, "Disable command execution")
	tellCmd.Flags().BoolVar(&autoExec, "auto-exec", false, "Automatically execute commands without confirmation when --apply is passed")

	tellCmd.Flags().Var(newAutoDebugValue(&autoDebug), "debug", "Automatically execute and debug failing commands (optionally specify number of tries—default is 5)")
	tellCmd.Flag("debug").NoOptDefVal = strconv.Itoa(defaultAutoDebugTries)
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
			return lib.CheckOutdatedContextWithOutput(false, autoConfirm || tellAutoApply || tellAutoContext, maybeContexts)
		},
	}, prompt, plan_exec.TellFlags{
		TellBg:      tellBg,
		TellStop:    tellStop,
		TellNoBuild: tellNoBuild,
		AutoContext: tellAutoContext,
		ExecEnabled: !noExec,
	})

	if tellAutoApply {
		flags := lib.ApplyFlags{
			AutoConfirm: true,
			AutoCommit:  autoCommit,
			NoCommit:    !autoCommit,
			NoExec:      noExec,
			AutoExec:    autoExec || autoDebug > 0,
			AutoDebug:   autoDebug,
		}

		lib.MustApplyPlan(
			lib.CurrentPlanId,
			lib.CurrentBranch,
			flags,
			plan_exec.GetOnApplyExecFail(flags),
		)
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
		term.OutputErrorAndExit("🚨 --apply can't be used with --no-build/-n")
	}
	if tellAutoApply && tellBg {
		term.OutputErrorAndExit("🚨 --apply can't be used with --bg")
	}
	if autoCommit && !tellAutoApply {
		term.OutputErrorAndExit("🚨 --commit/-c can only be used with --apply")
	}
	if autoExec && !tellAutoApply {
		term.OutputErrorAndExit("🚨 --auto-exec can only be used with --apply")
	}
	if autoDebug > 0 && !tellAutoApply {
		term.OutputErrorAndExit("🚨 --debug can only be used with --apply")
	}
	if autoDebug > 0 && noExec {
		term.OutputErrorAndExit("🚨 --debug can't be used with --no-exec")
	}

	if tellAutoContext && tellBg {
		term.OutputErrorAndExit("🚨 --auto-context/-c can't be used with --bg")
	}

}

func maybeShowDiffs() {
	diffs, err := api.Client.GetPlanDiffs(lib.CurrentPlanId, lib.CurrentBranch, plainTextOutput || showDiffUi)
	if err != nil {
		term.OutputErrorAndExit("Error getting plan diffs: %v", err)
		return
	}

	if len(diffs) > 0 {
		cmd := exec.Command(os.Args[0], "diffs", "--ui")

		// Create a context that's cancelled when the program exits
		ctx, cancel := context.WithCancel(context.Background())

		// Ensure cleanup on program exit
		go func() {
			// Wait for program exit signal
			c := make(chan os.Signal, 1)
			signal.Notify(c, os.Interrupt, syscall.SIGTERM)
			<-c

			// Cancel context and kill the process
			cancel()
			if cmd.Process != nil {
				cmd.Process.Kill()
			}
		}()

		go func() {
			if err := cmd.Start(); err != nil {
				fmt.Fprintf(os.Stderr, "Error starting diffs command: %v\n", err)
				return
			}

			// Wait in a separate goroutine
			go cmd.Wait()

			// Wait for either context cancellation or process completion
			<-ctx.Done()
			if cmd.Process != nil {
				cmd.Process.Kill()
			}
		}()

		// Give the UI a moment to start
		time.Sleep(100 * time.Millisecond)
	}
}

// AutoDebugValue implements the flag.Value interface
type autoDebugValue struct {
	value *int
}

func newAutoDebugValue(p *int) *autoDebugValue {
	*p = 0 // Default to 0 (disabled)
	return &autoDebugValue{p}
}

func (f *autoDebugValue) Set(s string) error {
	if s == "" {
		*f.value = defaultAutoDebugTries
		return nil
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return fmt.Errorf("invalid value for --debug: %v", err)
	}
	if v <= 0 {
		return fmt.Errorf("--debug value must be greater than 0")
	}
	*f.value = v
	return nil
}

func (f *autoDebugValue) String() string {
	if f.value == nil {
		return "0"
	}
	return strconv.Itoa(*f.value)
}

func (f *autoDebugValue) Type() string {
	return "int"
}
