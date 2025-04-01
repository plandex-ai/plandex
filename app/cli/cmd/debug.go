package cmd

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"plandex-cli/auth"
	"plandex-cli/lib"
	"plandex-cli/plan_exec"
	"plandex-cli/term"
	"plandex-cli/types"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

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
		// Use shell to handle operators like && and |
		shellCmdStr := "set -euo pipefail; " + cmdStr
		execCmd := exec.Command("sh", "-c", shellCmdStr)
		execCmd.Dir = cwd
		execCmd.Env = os.Environ()
		lib.SetPlatformSpecificAttrs(execCmd)

		pipe, err := execCmd.StdoutPipe()
		if err != nil {
			term.StopSpinner()
			term.OutputErrorAndExit("Failed to create pipe: %v", err)
		}
		execCmd.Stderr = execCmd.Stdout

		if err := execCmd.Start(); err != nil {
			term.StopSpinner()
			term.OutputErrorAndExit("Failed to start command: %v", err)
		}

		maybeDeleteCgroup := lib.MaybeIsolateCgroup(execCmd)

		ctx, cancel := context.WithCancel(context.Background())
		var interrupted atomic.Bool
		var interruptHandled atomic.Bool
		var interruptWG sync.WaitGroup

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)

		interruptWG.Add(1)
		go func() {
			defer interruptWG.Done()
			for {
				select {
				case sig := <-sigChan:
					if interruptHandled.CompareAndSwap(false, true) {
						fmt.Println()
						color.New(term.ColorHiYellow, color.Bold).Println("\n👉 Caught interrupt. Exiting gracefully...")
						interrupted.Store(true)

						var sysSig syscall.Signal

						switch sig {
						case os.Interrupt:
							// user pressed Ctrl+C
							sysSig = syscall.SIGINT
						case syscall.SIGTERM:
							// a polite "kill" request
							sysSig = syscall.SIGTERM
						case syscall.SIGHUP:
							sysSig = syscall.SIGHUP
						case syscall.SIGQUIT:
							sysSig = syscall.SIGQUIT
						default:
							sysSig = syscall.SIGINT
						}

						if err := lib.KillProcessGroup(execCmd, sysSig); err != nil {
							log.Printf("Failed to send signal %s to process group: %v", sysSig, err)
						}

						select {
						case <-time.After(2 * time.Second):
							color.New(term.ColorHiYellow, color.Bold).Println("👉 Commands didn't exit after 2 seconds. Sending SIGKILL.")
							if err := lib.KillProcessGroup(execCmd, syscall.SIGKILL); err != nil {
								log.Printf("Failed to send SIGKILL to process group: %v", err)
							}
							maybeDeleteCgroup()
						case <-ctx.Done():
							maybeDeleteCgroup()
							return
						}
					}
				case <-ctx.Done():
					maybeDeleteCgroup()
					return
				}
			}
		}()

		var outputBuilder strings.Builder
		scanner := bufio.NewScanner(pipe)
		go func() {
			for scanner.Scan() {
				line := scanner.Text()
				fmt.Println(line)
				outputBuilder.WriteString(line + "\n")
			}
		}()

		waitErr := execCmd.Wait()

		cancel()
		interruptWG.Wait()
		signal.Stop(sigChan)
		close(sigChan)

		if scanErr := scanner.Err(); scanErr != nil {
			log.Printf("⚠️ Scanner error reading subprocess output: %v", scanErr)
		}

		term.StopSpinner()

		outputStr := outputBuilder.String()
		if outputStr == "" && waitErr != nil {
			outputStr = waitErr.Error()
		}

		if outputStr != "" {
			fmt.Println(outputStr)
		}

		didSucceed := waitErr == nil

		if interrupted.Load() {
			color.New(term.ColorHiYellow, color.Bold).Println("👉  Execution interrupted")

			res, canceled, err := term.ConfirmYesNoCancel("Did the command succeed?")

			if err != nil {
				term.OutputErrorAndExit("Failed to get confirmation user input: %s", err)
			}

			didSucceed = res

			if canceled {
				os.Exit(0)
			}
		}

		if didSucceed {
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
		exitErr, ok := waitErr.(*exec.ExitError)
		status := -1
		if ok {
			status = exitErr.ExitCode()
		}

		prompt := fmt.Sprintf("'%s' failed with exit status %d. Output:\n\n%s\n\n--\n\n",
			strings.Join(cmdArgs, " "), status, outputStr)

		tellFlags := types.TellFlags{
			AutoContext: tellAutoContext,
			ExecEnabled: false,
			IsUserDebug: true,
		}

		plan_exec.TellPlan(plan_exec.ExecParams{
			CurrentPlanId: lib.CurrentPlanId,
			CurrentBranch: lib.CurrentBranch,
			ApiKeys:       apiKeys,
			CheckOutdatedContext: func(maybeContexts []*shared.Context, projectPaths *types.ProjectPaths) (bool, bool, error) {
				return lib.CheckOutdatedContextWithOutput(true, true, maybeContexts, projectPaths)
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
