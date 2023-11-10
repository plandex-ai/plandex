package cmd

import (
	"fmt"
	"os"
	"plandex/lib"
	"strconv"

	"github.com/spf13/cobra"
)

// rewindCmd represents the rewind command
var rewindCmd = &cobra.Command{
	Use:     "rewind [scope] [steps-or-sha]",
	Aliases: []string{"rw"},
	Short:   "Rewind the plan to an earlier state",
	Long: `Rewind the plan to an earlier state. Pass an optional scope to rewind only a specific part of the plan. Valid scopes are "convo", "context", and "draft". If no scope is passed, the entire plan is rewound.
	
	You can also optionally pass a "steps" number or commit sha. If a steps number is passed, the plan will be rewound that many steps. If a commit sha is passed, the plan will be rewound to that commit. If neither a steps number nor a commit sha is passed, the target scope will be rewound by 1 step.
	`,
	Args: cobra.MaximumNArgs(2),
	Run:  rewind,
}

var sha string

func init() {
	// Add rewind command
	RootCmd.AddCommand(rewindCmd)

	// Add sha flag
	rewindCmd.Flags().StringVar(&sha, "sha", "", "Specify a commit sha to rewind to")
}

func rewind(cmd *cobra.Command, args []string) {
	var dir string
	scope := "all"
	if len(args) > 0 {
		scope = args[0]
	}

	switch scope {
	case "convo":
		dir = lib.ConversationSubdir
	case "context":
		dir = lib.ContextSubdir
	case "draft":
		dir = lib.DraftSubdir
	default:
		dir = lib.CurrentPlanDir // Rewind the whole plan by default
	}

	// Check if either steps or sha is provided and not both
	stepsOrSHA := ""
	if len(args) > 1 {
		stepsOrSHA = args[1]
	}

	if steps, err := strconv.Atoi(stepsOrSHA); err == nil && steps > 0 {
		// Rewind by the specified number of steps
		if err := lib.GitRewindSteps(dir, steps); err != nil {
			fmt.Fprintln(os.Stderr, "Error rewinding steps:", err)
			os.Exit(1)
		}
	} else if sha := stepsOrSHA; sha != "" {
		// Rewind to the specified SHA
		if err := lib.GitRewindToSHA(dir, sha); err != nil {
			fmt.Fprintln(os.Stderr, "Error rewinding to SHA:", err)
			os.Exit(1)
		}
	} else if stepsOrSHA == "" {
		// No steps or SHA provided, rewind by 1 step by default
		if err := lib.GitRewindSteps(dir, 1); err != nil {
			fmt.Fprintln(os.Stderr, "Error rewinding 1 step:", err)
			os.Exit(1)
		}
	} else {
		fmt.Fprintln(os.Stderr, "Invalid steps or SHA. Steps must be a positive integer, and SHA must be a valid commit hash.")
		os.Exit(1)
	}
}
