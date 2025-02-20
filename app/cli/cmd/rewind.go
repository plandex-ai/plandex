package cmd

import (
	"fmt"
	"os"
	"plandex-cli/api"
	"plandex-cli/auth"
	"plandex-cli/fs"
	"plandex-cli/lib"
	"plandex-cli/term"
	"regexp"
	"strconv"
	"strings"
	"time"

	shared "plandex-shared"

	"github.com/spf13/cobra"
)

var rewindCmd = &cobra.Command{
	Use:     "rewind [steps-or-sha]",
	Aliases: []string{"rw"},
	Short:   "Rewind plan state and optionally revert project files",
	Long: `Rewind plan state and optionally revert project files to match.
	
You can pass a "steps" number or a commit sha. If a steps number is passed, 
the plan will be rewound that many steps. If a commit sha is passed, the 
plan will be rewound to that commit. If neither is passed, you will be 
prompted to select a step from the history.

By default, you will be prompted whether to revert project files to match 
the rewound plan state. You can use --revert to automatically revert 
files, or configure this behavior with the 'auto-revert' plan config setting.

If project files have changes, you will always be prompted before updating.`,
	Args: cobra.MaximumNArgs(1),
	Run:  rewind,
}

var revert bool
var skipRevert bool

func init() {
	RootCmd.AddCommand(rewindCmd)
	rewindCmd.Flags().BoolVar(&revert, "revert", false, "Also revert project files to match plan state")
	rewindCmd.Flags().BoolVar(&skipRevert, "skip-revert", false, "Skip reverting project files to match plan state")
	rewindCmd.Flags().BoolVar(&autoCommit, "commit", false, "Commit changes to git when --revert is passed")
	rewindCmd.Flags().BoolVar(&skipCommit, "skip-commit", false, "Skip committing changes to git when --revert is passed")

}

func rewind(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()
	lib.MustResolveProject()

	if lib.CurrentPlanId == "" {
		term.OutputNoCurrentPlanErrorAndExit()
	}

	if skipRevert && revert {
		term.OutputErrorAndExit("Cannot pass both --revert and --skip-revert")
	}

	// Get logs
	term.StartSpinner("")
	logsRes, apiErr := api.Client.ListLogs(lib.CurrentPlanId, lib.CurrentBranch)
	if apiErr != nil {
		term.StopSpinner()
		term.OutputErrorAndExit("Error getting logs: %v", apiErr)
	}

	var targetSha string
	var steps int
	var isSha bool

	parseGitLog := func(log string) (string, string) {
		// Parse the log entry
		lines := strings.Split(log, "\n")
		if len(lines) < 2 {
			return "", ""
		}

		// Extract sha and timestamp from first line
		parts := strings.Split(lines[0], "|")
		if len(parts) < 2 {
			return "", ""
		}

		shaLine := strings.TrimSpace(parts[0])
		sha := regexp.MustCompile(`📝 Update (\w+)`).FindStringSubmatch(shaLine)[1]
		msg := strings.TrimSpace(lines[1])

		return sha, msg
	}

	logEntries := strings.Split(logsRes.Body, "\n\n")

	if len(args) == 0 {
		// No arguments - show selection list
		options := make([]string, 0)

		for _, log := range logEntries {
			if log == "" {
				continue
			}

			sha, msg := parseGitLog(log)

			if sha == "" {
				continue
			}

			// Format the two-line option
			option := fmt.Sprintf("%s | %s", sha, formatLogMessage(msg))
			options = append(options, option)
		}

		term.StopSpinner()
		selected, err := term.SelectFromList("Select step to rewind to:", options)
		if err != nil {
			term.OutputErrorAndExit("Error selecting step: %v", err)
		}

		// Parse selected option to get sha
		parts := strings.Split(selected, " | ")
		if len(parts) < 2 {
			term.OutputErrorAndExit("Invalid selection")
		}

		targetSha = parts[0]
		isSha = true

	} else {
		// Arguments provided - use direct rewind logic
		stepsOrSha := args[0]
		steps, err := strconv.Atoi(stepsOrSha)

		if err == nil && steps > 0 && steps < 999 {
			// Rewind by the specified number of steps
			targetSha = logsRes.Shas[steps]
		} else if sha := stepsOrSha; sha != "" {
			// Rewind to the specified Sha
			targetSha = sha
			isSha = true
		} else {
			term.OutputErrorAndExit("Invalid steps or sha. Steps must be a positive integer, and sha must be a valid commit hash.")
		}
	}

	doRewind := func() {
		term.StartSpinner("")
		_, apiErr := api.Client.RewindPlan(lib.CurrentPlanId, lib.CurrentBranch, shared.RewindPlanRequest{
			Sha: targetSha,
		})
		term.StopSpinner()

		if apiErr != nil {
			term.OutputErrorAndExit("Error rewinding plan: %v", apiErr)
		}

		var msg string
		if isSha {
			msg = "✅ Rewound to " + targetSha
		} else {
			postfix := "s"
			if steps == 1 {
				postfix = ""
			}
			msg = fmt.Sprintf("✅ Rewound %d step%s to %s", steps, postfix, targetSha)
		}

		fmt.Println(msg)
		fmt.Println()
	}

	printNoChanges := func() {
		fmt.Println("🙅‍♂️ No project files were modified")
		fmt.Println()
	}

	printCmds := func() {
		term.PrintCmds("", "log", "continue")
	}

	// get the timestamp of the target sha
	var targetShaTimestamp time.Time
	for _, log := range logEntries {
		if log == "" {
			continue
		}
		sha, _ := parseGitLog(log)
		if sha == targetSha {
			timestamp, err := lib.GetGitLogTimestamp(log)
			if err != nil {
				continue
			}

			targetShaTimestamp = timestamp
			break
		}
	}

	if targetShaTimestamp.IsZero() {
		term.OutputErrorAndExit("Error getting timestamp for target sha: " + targetSha)
	}

	// Get current plan state to check for undone applies
	term.StartSpinner("")
	currentState, apiErr := api.Client.GetCurrentPlanState(lib.CurrentPlanId, lib.CurrentBranch)
	if apiErr != nil {
		term.StopSpinner()
		term.OutputErrorAndExit("Error getting plan state: %v", apiErr)
	}

	// Get list of applies that will be undone
	undonePlanApplies := lib.GetUndonePlanApplies(currentState, targetShaTimestamp)

	// If no applies are being undone, skip revert entirely
	if len(undonePlanApplies) == 0 {
		// Just do the rewind, no need for any file operations
		doRewind()
		printNoChanges()
		printCmds()
		return
	}

	// Get the set of affected file paths

	// Determine if we should revert based on flag/config
	var shouldRevert bool
	needsPrompt := true
	var config *shared.PlanConfig

	if cmd.Flags().Changed("revert") || cmd.Flags().Changed("skip-revert") {
		if skipRevert {
			shouldRevert = false
		} else {
			shouldRevert = revert
		}
		needsPrompt = false
	} else {
		config, apiErr = api.Client.GetPlanConfig(lib.CurrentPlanId)
		if apiErr != nil {
			term.OutputErrorAndExit("Error getting plan config: %v", apiErr)
		}
		shouldRevert = config.AutoRevertOnRewind
		needsPrompt = false
	}

	var targetState *shared.CurrentPlanState
	var analysis *lib.RewindAnalysis

	if shouldRevert || needsPrompt {
		// First preview the rewind to check for conflicts
		targetState, apiErr = api.Client.GetCurrentPlanStateAtSha(lib.CurrentPlanId, targetSha)
		if apiErr != nil {
			term.OutputErrorAndExit("Error previewing rewind: %v", apiErr)
		}

		if targetState == nil {
			term.OutputErrorAndExit("Error previewing rewind - no state found at sha: " + targetSha)
		}

		var err error
		analysis, err = lib.AnalyzeRewind(targetState, currentState)
		if err != nil {
			term.StopSpinner()
			term.OutputErrorAndExit("Error analyzing rewind: %v", err)
		}

		if len(analysis.RequiredChanges) == 0 {
			// No file changes - proceed with rewind
			doRewind()
			printNoChanges()
			printCmds()
			return
		}

		// Show file differences
		term.StopSpinner()

		// Group changes by type for display
		toAdd := make([]string, 0)
		toRemove := make([]string, 0)
		toModify := make([]string, 0)

		for path, content := range analysis.RequiredChanges {
			if content == "" {
				toRemove = append(toRemove, path)
			} else if currentState.ContextsByPath[path] == nil {
				toAdd = append(toAdd, path)
			} else {
				toModify = append(toModify, path)
			}
		}

		if needsPrompt || len(analysis.Conflicts) > 0 {
			s := "files"
			if len(analysis.RequiredChanges) == 1 {
				s = "file"
			}

			fmt.Printf("⏪ %d local project %s differ from target plan state\n", len(analysis.RequiredChanges), s)
			fmt.Println()
			fmt.Printf("Reverting the %s will make these changes locally 👇\n", s)
			fmt.Println()

			if len(toAdd) > 0 {
				fmt.Println("To add:")
				for _, path := range toAdd {
					fmt.Printf(" • %s\n", path)
				}
				fmt.Println()
			}

			if len(toRemove) > 0 {
				fmt.Println("To remove:")
				for _, path := range toRemove {
					fmt.Printf(" • %s\n", path)
				}
				fmt.Println()
			}

			if len(toModify) > 0 {
				fmt.Println("To update:")
				for _, path := range toModify {
					fmt.Printf(" • %s\n", path)
				}
				fmt.Println()
			}

			if len(analysis.Conflicts) > 0 {
				// Always prompt if there are conflicts
				s := " These project files have"
				if len(analysis.Conflicts) == 1 {
					s = " A project file has"
				}
				fmt.Printf("⚠️  %s been updated outside of Plandex since the latest apply:\n", s)
				for path := range analysis.Conflicts {
					fmt.Printf(" • %s\n", path)
				}
				fmt.Println()

				fmt.Println("If you revert, you will lose those changes.")
				fmt.Println()

				s = "files"
				if len(analysis.RequiredChanges) == 1 {
					s = "file"
				}

				options := []string{
					fmt.Sprintf("Revert project %s to match rewound plan state (overwrite changes)", s),
					fmt.Sprintf("Rewind plan, but skip reverting project %s", s),
					"Cancel rewind",
				}

				selected, err := term.SelectFromList("What do you want to do?", options)
				if err != nil {
					term.OutputErrorAndExit("Error getting user input: %v", err)
				}

				switch selected {
				case options[0]:
					shouldRevert = true
				case options[1]:
					shouldRevert = false
				case options[2]:
					os.Exit(0)
				}

				needsPrompt = false
			}
		}
	}

	// Now that we've handled the file state decision, perform the actual rewind
	doRewind()

	didRevert := false

	if shouldRevert || needsPrompt {
		if needsPrompt {
			term.StopSpinner()
			s := "files"
			if len(analysis.RequiredChanges) == 1 {
				s = "file"
			}
			confirmed, err := term.ConfirmYesNo(fmt.Sprintf("Revert project %s to match rewound plan state?", s))
			if err != nil {
				term.OutputErrorAndExit("Error getting user confirmation: %v", err)
			}

			shouldRevert = confirmed
		}

		if shouldRevert && len(analysis.RequiredChanges) > 0 {
			term.StartSpinner("")
			err := lib.ApplyRewindChanges(analysis.RequiredChanges)
			term.StopSpinner()
			if err != nil {
				term.OutputErrorAndExit("Error restoring file state: %v", err)
			}

			didRevert = true

			s := "files were"
			if len(analysis.RequiredChanges) == 1 {
				s = "file was"
			}

			fmt.Printf("⏪ %d project %s reverted\n", len(analysis.RequiredChanges), s)
			for path := range analysis.RequiredChanges {
				fmt.Printf(" • %s\n", path)
			}
			fmt.Println()
		}
	}

	if didRevert {
		shouldCommit := false
		needsPrompt := true

		if cmd.Flags().Changed("commit") || cmd.Flags().Changed("skip-commit") {
			if skipCommit {
				shouldCommit = false
			} else {
				shouldCommit = autoCommit
			}
			needsPrompt = false
		} else {
			if config == nil {
				config, apiErr = api.Client.GetPlanConfig(lib.CurrentPlanId)
				if apiErr != nil {
					term.OutputErrorAndExit("Error getting plan config: %v", apiErr)
				}
			}
			shouldCommit = config.AutoCommit
			needsPrompt = false
		}

		if needsPrompt {
			term.StopSpinner()
			confirmed, err := term.ConfirmYesNo("Commit changes to git?")
			if err != nil {
				term.OutputErrorAndExit("Error getting user confirmation: %v", err)
			}
			shouldCommit = confirmed
		}

		if shouldCommit {
			msg := "🤖 Plandex → rewound plan state and reverted these changes:"
			for _, apply := range undonePlanApplies {
				msg += fmt.Sprintf("\n   • %s", apply.CommitMsg)
			}

			paths := []string{}
			for path := range analysis.RequiredChanges {
				paths = append(paths, path)
			}

			err := lib.GitAddAndCommitPaths(fs.ProjectRoot, msg, paths, true)
			if err != nil {
				term.OutputErrorAndExit("Error committing changes: %v", err)
			}
		}
	} else {
		printNoChanges()
	}

	printCmds()
}

func formatLogMessage(msg string) string {
	var res string

	// Check for message type patterns
	switch {
	case strings.Contains(msg, "User prompt"):
		res = "💬 " + msg
	case strings.Contains(msg, "Plandex reply"):
		if coins := regexp.MustCompile(`(\d+) 🪙`).FindStringSubmatch(msg); len(coins) >= 2 {
			res = "🤖 AI Response | " + coins[1] + " 🪙"
		}
		res = "🤖 " + msg
	case strings.Contains(msg, "Build pending"):
		res = "🏗️  Building changes"
	case strings.Contains(msg, "Loaded"):
		res = "📚 " + msg
	default:
		res = msg
	}

	return res
}
