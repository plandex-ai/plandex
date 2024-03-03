package cmd

import (
	"fmt"
	"plandex/api"
	"plandex/auth"
	"plandex/lib"
	"plandex/term"
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

func init() {
	// Add rewind command
	RootCmd.AddCommand(rewindCmd)
}

func rewind(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()
	lib.MustResolveProject()

	if lib.CurrentPlanId == "" {
		fmt.Println("🤷‍♂️ No current plan")
		return
	}

	var stepsOrSha string
	if len(args) > 0 {
		stepsOrSha = args[0]
	} else {
		stepsOrSha = "1"
	}

	term.StartSpinner("")
	logsRes, apiErr := api.Client.ListLogs(lib.CurrentPlanId, lib.CurrentBranch)

	if apiErr != nil {
		term.OutputErrorAndExit("Error getting logs: %v", apiErr)
	}

	var targetSha string

	// log.Println("shas:", logsRes.Shas)

	steps, err := strconv.Atoi(stepsOrSha)
	isSha := false

	if err == nil && steps > 0 && steps < 999 {
		// log.Println("steps:", steps)
		// Rewind by the specified number of steps
		targetSha = logsRes.Shas[steps]
	} else if sha := stepsOrSha; sha != "" {
		// log.Println("sha provided:", sha)
		// Rewind to the specified Sha
		targetSha = sha
		isSha = true
	} else if stepsOrSha == "" {
		// log.Println("No steps or sha provided, rewinding by 1 step")
		// Rewind by 1 step
		steps = 1
		targetSha = logsRes.Shas[1]
	} else {
		term.OutputErrorAndExit("Invalid steps or sha. Steps must be a positive integer, and sha must be a valid commit hash.")
	}

	// log.Println("Rewinding to", targetSha)

	// Rewind to the target sha
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

	term.PrintCmds("", "log")

	// fmt.Println(rwRes.LatestCommit)
}
