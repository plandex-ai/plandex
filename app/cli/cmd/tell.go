package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"plandex-cli/auth"
	"plandex-cli/lib"
	"plandex-cli/plan_exec"
	"plandex-cli/term"
	"plandex-cli/types"
	"strings"

	shared "plandex-shared"

	"github.com/spf13/cobra"
)

var isImplementationOfChat bool

// tellCmd represents the prompt command
var tellCmd = &cobra.Command{
	Use:     "tell [prompt]",
	Aliases: []string{"t"},
	Short:   "Send a prompt for the current plan",
	// Long:  ``,
	Args: cobra.RangeArgs(0, 1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{}, cobra.ShellCompDirectiveNoFileComp
	},
	Run: doTell,
}

func init() {
	RootCmd.AddCommand(tellCmd)

	initExecFlags(tellCmd, initExecFlagsParams{})

	tellCmd.Flags().BoolVar(&isImplementationOfChat, "from-chat", false, "Begin implementation based on conversation so far")
}

func doTell(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()
	lib.MustResolveProject()
	mustSetPlanExecFlags(cmd, false)

	if isImplementationOfChat && len(args) > 0 {
		term.OutputErrorAndExit("Error: --from-chat cannot be used with a prompt")
	}

	var prompt string
	if !isImplementationOfChat {
		prompt = getTellPrompt(args)

		if prompt == "" {
			fmt.Println("🤷‍♂️ No prompt to send")
			return
		}
	}

	tellFlags := types.TellFlags{
		TellBg:                 tellBg,
		TellStop:               tellStop,
		TellNoBuild:            tellNoBuild,
		AutoContext:            tellAutoContext,
		SmartContext:           tellSmartContext,
		ExecEnabled:            !noExec,
		AutoApply:              tellAutoApply,
		IsImplementationOfChat: isImplementationOfChat,
		SkipChangesMenu:        tellSkipMenu,
	}

	plan_exec.TellPlan(plan_exec.ExecParams{
		CurrentPlanId: lib.CurrentPlanId,
		CurrentBranch: lib.CurrentBranch,
		AuthVars:      lib.MustVerifyAuthVars(auth.Current.IntegratedModelsMode),
		CheckOutdatedContext: func(maybeContexts []*shared.Context, projectPaths *types.ProjectPaths) (bool, bool, error) {
			auto := autoConfirm || tellAutoApply || tellAutoContext
			return lib.CheckOutdatedContextWithOutput(auto, auto, maybeContexts, projectPaths)
		},
	}, prompt, tellFlags)

	if tellAutoApply {
		applyFlags := types.ApplyFlags{
			AutoConfirm: true,
			AutoCommit:  autoCommit,
			NoCommit:    !autoCommit,
			NoExec:      noExec,
			AutoExec:    autoExec || autoDebug > 0,
			AutoDebug:   autoDebug,
		}

		lib.MustApplyPlan(lib.ApplyPlanParams{
			PlanId:     lib.CurrentPlanId,
			Branch:     lib.CurrentBranch,
			ApplyFlags: applyFlags,
			TellFlags:  tellFlags,
			OnExecFail: plan_exec.GetOnApplyExecFail(applyFlags, tellFlags),
		})
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

func getEditorInstructions() string {
	if editor == EditorTypeVim {
		return "👉  Write your prompt below, then save and exit to send it to Plandex.\n• To save and exit, press ESC, then type :wq! and press ENTER.\n• To exit without saving, press ESC, then type :q! and press ENTER.\n\n\n"
	}

	if editor == EditorTypeNano {
		return "👉  Write your prompt below, then save and exit to send it to Plandex.\n• To save and exit, press Ctrl+X, then Y, then ENTER.\n• To exit without saving, press Ctrl+X, then N.\n\n\n"
	}

	return "👉  Write your prompt below, then save and exit to send it to Plandex.\n\n\n"
}

func getEditorPrompt() string {
	tempFile, err := os.CreateTemp(os.TempDir(), "plandex_prompt_*")
	if err != nil {
		term.OutputErrorAndExit("Failed to create temporary file: %v", err)
	}

	instructions := getEditorInstructions()
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

// func maybeShowDiffs() {
// 	diffs, err := api.Client.GetPlanDiffs(lib.CurrentPlanId, lib.CurrentBranch, plainTextOutput || showDiffUi)
// 	if err != nil {
// 		term.OutputErrorAndExit("Error getting plan diffs: %v", err)
// 		return
// 	}

// 	if len(diffs) > 0 {
// 		cmd := exec.Command(os.Args[0], "diffs", "--ui")

// 		// Create a context that's cancelled when the program exits
// 		ctx, cancel := context.WithCancel(context.Background())

// 		// Ensure cleanup on program exit
// 		go func() {
// 			// Wait for program exit signal
// 			c := make(chan os.Signal, 1)
// 			signal.Notify(c, os.Interrupt, syscall.SIGTERM)
// 			<-c

// 			// Cancel context and kill the process
// 			cancel()
// 			if cmd.Process != nil {
// 				cmd.Process.Kill()
// 			}
// 		}()

// 		go func() {
// 			if err := cmd.Start(); err != nil {
// 				fmt.Fprintf(os.Stderr, "Error starting diffs command: %v\n", err)
// 				return
// 			}

// 			// Wait in a separate goroutine
// 			go cmd.Wait()

// 			// Wait for either context cancellation or process completion
// 			<-ctx.Done()
// 			if cmd.Process != nil {
// 				cmd.Process.Kill()
// 			}
// 		}()

// 		// Give the UI a moment to start
// 		time.Sleep(100 * time.Millisecond)
// 	}
// }
