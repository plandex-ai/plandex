package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"plandex-cli/auth"
	"plandex-cli/lib"
	"plandex-cli/plan_exec"
	"plandex-cli/term"
	"plandex-cli/types"
	"strconv"
	"strings"

	shared "plandex-shared"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

const DebugDefaultTries = 5

var debugCmd = &cobra.Command{
	Use:     "debug [tries] <cmd>",
	Aliases: []string{"db"},
	Short:   "Debug a failing command with Plandex",
	Args:    cobra.MinimumNArgs(1),
	Run:     doDebug,
}

func init() {
	RootCmd.AddCommand(debugCmd)
	debugCmd.Flags().BoolVarP(&autoCommit, "commit", "c", false, "Commit changes after successful execution")
}

func doDebug(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()
	lib.MustResolveProject()
	mustSetPlanExecFlags(cmd)

	if lib.CurrentPlanId == "" {
		term.OutputNoCurrentPlanErrorAndExit()
	}

	// Parse tries and command
	tries := DebugDefaultTries
	cmdArgs := args

	// Check if first arg is tries count
	if val, err := strconv.Atoi(args[0]); err == nil {
		if val <= 0 {
			term.OutputErrorAndExit("Tries must be greater than 0")
		}
		tries = val
		cmdArgs = args[1:]
		if len(cmdArgs) == 0 {
			term.OutputErrorAndExit("No command specified")
		}
	}

	var apiKeys map[string]string
	if !auth.Current.IntegratedModelsMode {
		apiKeys = lib.MustVerifyApiKeys()
	}

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		term.OutputErrorAndExit("Failed to get working directory: %v", err)
	}

	cmdStr := strings.Join(cmdArgs, " ")

	// Execute command and handle retries
	for attempt := 0; attempt < tries; attempt++ {
		term.StartSpinner("")
		// Use shell to handle operators like && and |
		execCmd := exec.Command("sh", "-c", cmdStr)
		execCmd.Dir = cwd
		execCmd.Env = os.Environ()

		output, err := execCmd.CombinedOutput()

		term.StopSpinner()

		outputStr := string(output)
		if outputStr == "" && err != nil {
			// If no output but error occurred, include error in output
			outputStr = err.Error()
		}

		if outputStr != "" {
			fmt.Println(outputStr)
		}

		if err == nil {
			if attempt == 0 {
				fmt.Printf("✅ Command %s succeeded on first try\n", color.New(color.Bold, term.ColorHiCyan).Sprintf(cmdStr))
			} else {
				lbl := "attempts"
				if attempt == 1 {
					lbl = "attempt"
				}
				fmt.Printf("✅ Command %s succeeded after %d fix %s\n", color.New(color.Bold, term.ColorHiCyan).Sprintf(cmdStr), attempt, lbl)
			}
			return
		}

		if attempt == tries-1 {
			fmt.Printf("Command failed after %d tries\n", tries)
			os.Exit(1)
		}

		// Prepare prompt for TellPlan
		exitErr, ok := err.(*exec.ExitError)
		status := -1
		if ok {
			status = exitErr.ExitCode()
		}

		prompt := fmt.Sprintf("'%s' failed with exit status %d. Output:\n\n%s\n\n--\n\n",
			strings.Join(cmdArgs, " "), status, string(output))

		tellFlags := types.TellFlags{
			AutoContext: tellAutoContext,
			ExecEnabled: false,
			IsUserDebug: true,
		}

		plan_exec.TellPlan(plan_exec.ExecParams{
			CurrentPlanId: lib.CurrentPlanId,
			CurrentBranch: lib.CurrentBranch,
			ApiKeys:       apiKeys,
			CheckOutdatedContext: func(maybeContexts []*shared.Context) (bool, bool, error) {
				return lib.CheckOutdatedContextWithOutput(true, true, maybeContexts)
			},
		}, prompt, tellFlags)

		applyFlags := types.ApplyFlags{
			AutoConfirm: true,
			AutoCommit:  autoCommit,
			NoCommit:    !autoCommit,
			NoExec:      false,
			AutoExec:    true,
		}

		lib.MustApplyPlan(lib.ApplyPlanParams{
			PlanId:      lib.CurrentPlanId,
			Branch:      lib.CurrentBranch,
			ApplyFlags:  applyFlags,
			TellFlags:   tellFlags,
			OnExecFail:  plan_exec.GetOnApplyExecFailWithCommand(applyFlags, tellFlags, cmdStr),
			ExecCommand: cmdStr,
		})
	}
}
