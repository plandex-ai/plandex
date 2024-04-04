package main

import (
	"log"
	"os"
	"path/filepath"
	"plandex/api"
	"plandex/auth"
	"plandex/cmd"
	"plandex/fs"
	"plandex/lib"
	"plandex/plan_exec"
	"plandex/term"

	"github.com/plandex/plandex/shared"
)

func init() {
	// inter-package dependency injections to avoid circular imports
	auth.SetApiClient(api.Client)
	lib.SetBuildPlanInlineFn(func(maybeContexts []*shared.Context) (bool, error) {
		return plan_exec.Build(plan_exec.ExecParams{
			CurrentPlanId: lib.CurrentPlanId,
			CurrentBranch: lib.CurrentBranch,
			CheckOutdatedContext: func(maybeContexts []*shared.Context) (bool, bool) {
				return lib.MustCheckOutdatedContext(true, maybeContexts)
			},
		}, false)
	})

	// set up a file logger
	// TODO: log rotation

	file, err := os.OpenFile(filepath.Join(fs.HomePlandexDir, "plandex.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		term.OutputErrorAndExit("Error opening log file: %v", err)
	}

	// Set the output of the logger to the file
	log.SetOutput(file)

	// log.Println("Starting Plandex - logging initialized")
}

func main() {
	checkForUpgrade()

	// Manually check for help flags at the root level
	if len(os.Args) == 2 && (os.Args[1] == "-h" || os.Args[1] == "--help") {
		// Display your custom help here
		term.PrintCustomHelp()
		os.Exit(0)
	}

	cmd.Execute()
}
