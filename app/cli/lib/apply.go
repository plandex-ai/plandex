package lib

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"plandex/api"
	"plandex/auth"
	"plandex/fs"
	"plandex/term"
	"strings"
	"syscall"

	"github.com/fatih/color"
	"github.com/plandex/plandex/shared"
)

type ApplyFlags struct {
	AutoConfirm bool
	AutoCommit  bool
	NoCommit    bool
	AutoExec    bool
	NoExec      bool
	AutoDebug   int
}

type onErrFn func(errMsg string, errArgs ...interface{})

func MustApplyPlan(
	planId,
	branch string,
	flags ApplyFlags,
	onExecFail func(status int, output string, attempt int),
) {
	MustApplyPlanAttempt(planId, branch, flags, onExecFail, 0)
}

func MustApplyPlanAttempt(
	planId,
	branch string,
	flags ApplyFlags,
	onExecFail func(status int, output string, attempt int),
	attempt int,
) {
	log.Println("Applying plan")

	autoConfirm := flags.AutoConfirm
	autoCommit := flags.AutoCommit
	noCommit := flags.NoCommit
	noExec := flags.NoExec

	term.StartSpinner("")

	currentPlanState, apiErr := api.Client.GetCurrentPlanState(planId, branch)

	if apiErr != nil {
		term.StopSpinner()
		term.OutputErrorAndExit("Error getting current plan state: %v", apiErr)
	}

	if currentPlanState.HasPendingBuilds() {
		plansRunningRes, apiErr := api.Client.ListPlansRunning([]string{CurrentProjectId}, false)

		if apiErr != nil {
			term.StopSpinner()
			term.OutputErrorAndExit("Error getting running plans: %v", apiErr)
		}

		for _, b := range plansRunningRes.Branches {
			if b.PlanId == planId && b.Name == branch {
				fmt.Println("This plan is currently active. Please wait for it to finish before applying.")
				fmt.Println()
				term.PrintCmds("", "ps", "connect")
				return
			}
		}

		term.StopSpinner()

		fmt.Println("This plan has changes that need to be built before applying")
		fmt.Println()

		shouldBuild, err := term.ConfirmYesNo("Build changes now?")

		if err != nil {
			term.OutputErrorAndExit("failed to get confirmation user input: %s", err)
		}

		if !shouldBuild {
			fmt.Println("Apply plan canceled")
			os.Exit(0)
		}

		_, err = buildPlanInlineFn(autoConfirm, nil)

		if err != nil {
			term.OutputErrorAndExit("failed to build plan: %v", err)
		}
	}

	anyOutdated, didUpdate, err := CheckOutdatedContextWithOutput(true, autoConfirm, nil)

	if err != nil {
		term.OutputErrorAndExit("error checking outdated context: %v", err)
	}

	if anyOutdated && !didUpdate {
		term.StopSpinner()
		fmt.Println("Apply plan canceled")
		os.Exit(0)
	}

	currentPlanFiles := currentPlanState.CurrentPlanFiles
	isRepo := fs.ProjectRootIsGitRepo()

	toApply := currentPlanFiles.Files
	hasExec := currentPlanFiles.Files["_apply.sh"] != ""

	log.Printf("Files to apply: %d, Has exec script: %v", len(toApply), hasExec)

	if len(toApply) == 0 {
		term.StopSpinner()
		fmt.Println("🤷‍♂️ No changes to apply")
		return
	}

	hasFileChanges := !hasExec || len(toApply) > 1

	onErr := func(errMsg string, errArgs ...interface{}) {
		term.StopSpinner()
		term.OutputErrorAndExit(errMsg, errArgs...)
	}

	onGitErr := func(errMsg, unformattedErrMsg string) {
		term.StopSpinner()
		term.OutputSimpleError(errMsg, unformattedErrMsg)
	}

	log.Println("Getting API keys")

	var apiKeys map[string]string
	if !auth.Current.IntegratedModelsMode {
		apiKeys = MustVerifyApiKeysSilent()
	}

	log.Println("Has file changes:", hasFileChanges)

	if hasFileChanges {
		if !autoConfirm {
			log.Println("Asking user to confirm applying changes")

			term.StopSpinner()
			numToApply := len(toApply)
			suffix := ""
			if numToApply > 1 {
				suffix = "s"
			}
			shouldContinue, err := term.ConfirmYesNo("Apply changes to %d file%s?", numToApply, suffix)

			if err != nil {
				term.OutputErrorAndExit("failed to get confirmation user input: %s", err)
			}

			if !shouldContinue {
				os.Exit(0)
			}
			term.ResumeSpinner()
		}

		var commitSummary string

		openAIBase := os.Getenv("OPENAI_API_BASE")
		if openAIBase == "" {
			openAIBase = os.Getenv("OPENAI_ENDPOINT")
		}

		log.Println("Applying plan with API call")

		commitSummary, apiErr = api.Client.ApplyPlan(planId, branch, shared.ApplyPlanRequest{
			ApiKeys:     apiKeys,
			OpenAIBase:  openAIBase,
			OpenAIOrgId: os.Getenv("OPENAI_ORG_ID"),
		})

		if apiErr != nil {
			onErr("failed to set pending results applied: %s", apiErr.Msg)
			return
		}

		log.Println("Applying plan files")

		var updatedFiles []string
		for path, content := range toApply {
			if path == "_apply.sh" {
				continue
			}

			// Compute destination path
			dstPath := filepath.Join(fs.ProjectRoot, path)

			content = strings.ReplaceAll(content, "\\`\\`\\`", "```")

			// Check if the file exists
			var exists bool
			_, err := os.Stat(dstPath)
			if err == nil {
				exists = true
			} else {
				if os.IsNotExist(err) {
					exists = false
				} else {
					onErr("failed to check if %s exists:", dstPath)
					return
				}
			}

			if exists {
				// read file content
				bytes, err := os.ReadFile(dstPath)

				if err != nil {
					onErr("failed to read %s:", dstPath)
					return
				}

				// Check if the file has changed
				if string(bytes) == content {
					// log.Println("File is unchanged, skipping")
					continue
				} else {
					updatedFiles = append(updatedFiles, path)
				}
			} else {
				updatedFiles = append(updatedFiles, path)

				// Create the directory if it doesn't exist
				err := os.MkdirAll(filepath.Dir(dstPath), 0755)
				if err != nil {
					onErr("failed to create directory %s:", filepath.Dir(dstPath))
					return
				}
			}

			// Write the file
			err = os.WriteFile(dstPath, []byte(content), 0644)
			if err != nil {
				onErr("failed to write %s:", dstPath)
				return
			}
		}

		term.StopSpinner()

		log.Println("Applying plan files complete")

		if len(updatedFiles) == 0 {
			fmt.Println("✅ Applied changes, but no files were updated")
			return
		} else {
			if isRepo && !noCommit {
				confirmed := autoCommit
				if !autoCommit {
					fmt.Println("✏️  Plandex can commit these updates with an automatically generated message.")
					fmt.Println()
					// fmt.Println("ℹ️  Only the files that Plandex is updating will be included the commit. Any other changes, staged or unstaged, will remain exactly as they are.")
					// fmt.Println()

					confirmed, err = term.ConfirmYesNo("Commit Plandex updates now?")

					if err != nil {
						onErr("failed to get confirmation user input: %s", err)
					}
				}

				if confirmed {
					// Commit the changes
					msg := currentPlanState.PendingChangesSummaryForApply(commitSummary)

					// log.Println("Committing changes with message:")
					// log.Println(msg)

					// spew.Dump(currentPlanState)

					err := GitAddAndCommitPaths(fs.ProjectRoot, msg, updatedFiles, true)
					if err != nil {
						onGitErr("Failed to commit changes:", err.Error())
					}
				}
			}

			suffix := ""
			if len(updatedFiles) > 1 {
				suffix = "s"
			}
			fmt.Printf("✅ Applied changes, %d file%s updated\n", len(updatedFiles), suffix)
		}
	}

	term.StopSpinner()

	if _, ok := toApply["_apply.sh"]; ok && !noExec {
		handleApplyScript(flags, toApply, onErr, onExecFail, attempt)
	}
}

func handleApplyScript(
	flags ApplyFlags,
	toApply map[string]string,
	onErr onErrFn,
	onExecFail func(status int, output string, attempt int),
	attempt int,
) {
	log.Println("Handling apply script")

	color.New(term.ColorHiCyan, color.Bold).Println("🚀 Commands to execute 👇")

	content := toApply["_apply.sh"]

	md, err := term.GetMarkdown("```bash\n" + content + "\n```")

	if err != nil {
		onErr("failed to get markdown representation: %s", err)
	}

	fmt.Println(strings.TrimSpace(md))

	log.Println("Asking user to confirm executing apply script")

	var confirmed bool
	if flags.AutoExec {
		confirmed = true
	} else {
		confirmed, err = term.ConfirmYesNo("Execute now?")

		if err != nil {
			onErr("failed to get confirmation user input: %s", err)
		}
	}

	if confirmed {
		log.Println("Executing apply script")
		execApplyScript(toApply, onErr, onExecFail, attempt)
	}
}

var shellShebangs = map[string]string{
	"/bin/bash": `#!/bin/bash
`,
	"/bin/zsh": `#!/bin/zsh
`,
}

var applyScriptErrorHandling = map[string]string{
	"/bin/bash": `set -euo pipefail
trap 'echo "Error on line $LINENO: $BASH_COMMAND"' ERR
`,
	"/bin/zsh": `set -euo pipefail
trap 'echo "Error on line $LINENO: ${funcfiletrace[0]#*:}"' ERR
`,
}

func execApplyScript(
	toApply map[string]string,
	onErr onErrFn,
	onExecFail func(status int, output string, attempt int),
	attempt int,
) {
	log.Println("Executing apply script")

	color.New(term.ColorHiCyan, color.Bold).Println("🚀 Executing... Output below:")
	fmt.Println()

	dstPath := filepath.Join(fs.ProjectRoot, "_apply.sh")

	content := toApply["_apply.sh"]
	lines := strings.Split(content, "\n")
	filteredLines := []string{}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#!/") {
			continue
		}
		if strings.HasPrefix(trimmed, "set -") || strings.HasSuffix(trimmed, "pipefail") {
			continue
		}
		if strings.HasPrefix(trimmed, "trap") {
			continue
		}
		filteredLines = append(filteredLines, line)
	}

	// Detect shell
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash" // fallback
	}

	// Get appropriate header
	shebang := shellShebangs[shell]
	if shebang == "" {
		shebang = shellShebangs["/bin/bash"] // fallback if shell not supported
	}
	errorHandling := applyScriptErrorHandling[shell]

	if errorHandling == "" {
		errorHandling = applyScriptErrorHandling["/bin/bash"] // fallback if shell not supported
	}

	header := shebang + "\n" + errorHandling
	content = header + "\n" + strings.Join(filteredLines, "\n")
	err := os.WriteFile(dstPath, []byte(content), 0755)

	if err != nil {
		onErr("failed to write _apply.sh: %s", err)
	}

	execCmd := exec.Command(shell, "-l", dstPath)
	execCmd.Dir = fs.ProjectRoot
	execCmd.Env = os.Environ()
	execCmd.Stdin = os.Stdin

	// Create a pipe for both stdout and stderr
	pipe, err := execCmd.StdoutPipe()
	if err != nil {
		os.Remove(dstPath)
		onErr("failed to create stdout pipe: %s", err)
	}
	execCmd.Stderr = execCmd.Stdout

	if err := execCmd.Start(); err != nil {
		os.Remove(dstPath)
		onErr("failed to start command: %s", err)
	}

	// Handle SIGINT and SIGTERM -- delete _apply.sh and kill process
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		os.Remove(dstPath)
		execCmd.Process.Signal(sig)
	}()
	defer signal.Stop(sigChan)

	// Read and display output in real-time
	scanner := bufio.NewScanner(pipe)
	var outputBuilder strings.Builder
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Println(line)
		outputBuilder.WriteString(line + "\n")
	}

	err = execCmd.Wait()

	// remove _apply.sh without overwriting err val
	{
		err := os.Remove(dstPath)
		if err != nil && !os.IsNotExist(err) {
			onErr("failed to remove _apply.sh: %s", err)
		}
	}

	if err != nil {
		fmt.Println()
		color.New(term.ColorHiRed, color.Bold).Println("🚨 Commands failed")

		exitErr, ok := err.(*exec.ExitError)
		status := -1
		if ok {
			status = exitErr.ExitCode()
		}
		onExecFail(status, outputBuilder.String(), attempt)
	} else {
		fmt.Println()
		fmt.Println("✅ Commands succeeded")
	}

}
