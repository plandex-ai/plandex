package cmd

import (
	"fmt"
	"os"
	"plandex/api"
	"plandex/lib"
	"strconv"

	"github.com/plandex/plandex/shared"
	"github.com/spf13/cobra"
)

// rewindCmd represents the rewind command
var rewindCmd = &cobra.Command{
	Use:     "rewind [steps-or-sha]",
	Aliases: []string{"rw"},
	Short:   "Rewind the plan to an earlier state",
	Long: `Rewind the plan to an earlier state.
	
	You can pass a "steps" number or a commit sha. If a steps number is passed, the plan will be rewound that many steps. If a commit sha is passed, the plan will be rewound to that commit. If neither a steps number nor a commit sha is passed, the target scope will be rewound by 1 step.
	`,
	Args: cobra.MaximumNArgs(1),
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

	// Check if either steps or sha is provided and not both
	stepsOrSha := ""
	if len(args) > 1 {
		stepsOrSha = args[1]
	}

	logsRes, err := api.Client.ListLogs(lib.CurrentPlanId)

	if err != nil {
		fmt.Printf("Error getting logs: %v\n", err)
		return
	}

	var targetSha string

	if steps, err := strconv.Atoi(stepsOrSha); err == nil && steps > 0 {
		// Rewind by the specified number of steps
		targetSha = logsRes.Shas[steps]
	} else if sha := stepsOrSha; sha != "" {
		// Rewind to the specified Sha
		targetSha = sha
	} else if stepsOrSha == "" {
		// Rewind by 1 step
		targetSha = logsRes.Shas[1]
	} else {
		fmt.Fprintln(os.Stderr, "Invalid steps or sha. Steps must be a positive integer, and sha must be a valid commit hash.")
		os.Exit(1)
	}

	// Rewind to the target sha
	rwRes, err := api.Client.RewindPlan(lib.CurrentPlanId, shared.RewindPlanRequest{Sha: targetSha})

	msg := "✅ Rewound "

	msg += "to " + targetSha

	fmt.Println(msg)
	fmt.Println()
	fmt.Println(rwRes.LatestCommit)
}
