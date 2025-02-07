package cmd

import (
	"fmt"
	"plandex-cli/api"
	"plandex-cli/auth"
	"plandex-cli/lib"
	"plandex-cli/term"
	"regexp"
	"strconv"
	"strings"

	shared "plandex-shared"

	"github.com/spf13/cobra"
)

var rewindCmd = &cobra.Command{
	Use:     "rewind [steps-or-sha]",
	Aliases: []string{"rw"},
	Short:   "Rewind the plan to an earlier state",
	Long: `Rewind the plan to an earlier state.
	
	You can pass a "steps" number or a commit sha. If a steps number is passed, the plan will be rewound that many steps. If a commit sha is passed, the plan will be rewound to that commit. If neither a steps number nor a commit sha is passed, you will be prompted to select a step from the history.`,
	Args: cobra.MaximumNArgs(1),
	Run:  rewind,
}

func init() {
	RootCmd.AddCommand(rewindCmd)
}

func rewind(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()
	lib.MustResolveProject()

	if lib.CurrentPlanId == "" {
		term.OutputNoCurrentPlanErrorAndExit()
	}

	// Get logs
	term.StartSpinner("")
	logsRes, apiErr := api.Client.ListLogs(lib.CurrentPlanId, lib.CurrentBranch)
	if apiErr != nil {
		term.StopSpinner()
		term.OutputErrorAndExit("Error getting logs: %v", apiErr)
	}
	term.StopSpinner()

	var targetSha string
	var steps int
	var isSha bool

	if len(args) == 0 {
		// No arguments - show selection list
		options := make([]string, 0)
		logEntries := strings.Split(logsRes.Body, "\n\n")

		for _, log := range logEntries {
			if log == "" {
				continue
			}

			// Parse the log entry
			lines := strings.Split(log, "\n")
			if len(lines) < 2 {
				continue
			}

			// Extract sha and timestamp from first line
			// Format: 📝 Update 526c4a9[0;22m | [36mToday | 2:35:18am PST[0m
			parts := strings.Split(lines[0], "|")
			if len(parts) < 2 {
				continue
			}

			shaLine := strings.TrimSpace(parts[0])
			sha := regexp.MustCompile(`📝 Update (\w+)`).FindStringSubmatch(shaLine)[1]

			// Get message from second line and format it
			msg := strings.TrimSpace(lines[1])

			// Format the two-line option
			option := fmt.Sprintf("%s | %s", sha, formatLogMessage(msg))
			options = append(options, option)
		}

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

	// Rewind to the target sha
	term.StartSpinner("")
	_, apiErr = api.Client.RewindPlan(lib.CurrentPlanId, lib.CurrentBranch, shared.RewindPlanRequest{Sha: targetSha})
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

	term.PrintCmds("", "log", "continue")
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
