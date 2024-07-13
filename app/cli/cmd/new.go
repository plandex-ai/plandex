package cmd

import (
	"fmt"
	"os"

	"plandex/api"
	"plandex/auth"
	"plandex/lib"
	"plandex/term"

	"github.com/fatih/color"
	"github.com/plandex/plandex/shared"
	"github.com/spf13/cobra"
)

var name string

// newCmd represents the new command
var newCmd = &cobra.Command{
	Use:     "new",
	Aliases: []string{"n"},
	Short:   "Start a new plan",
	// Long:  ``,
	Args: cobra.ExactArgs(0),
	Run:  new,
}

func init() {
	RootCmd.AddCommand(newCmd)
	newCmd.Flags().StringVarP(&name, "name", "n", "", "Name of the new plan")
}

func new(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()
	lib.MustResolveOrCreateProject()

	term.StartSpinner("")
	res, apiErr := api.Client.CreatePlan(lib.CurrentProjectId, shared.CreatePlanRequest{Name: name})
	term.StopSpinner()

	if apiErr != nil {
		if apiErr.Type == shared.ApiErrorTypeTrialPlansExceeded {
			fmt.Fprintf(os.Stderr, "🚨 You've reached the Plandex Cloud anonymous trial limit of %d plans\n", apiErr.TrialPlansExceededError.MaxPlans)

			res, err := term.ConfirmYesNo("Upgrade to an unlimited free account?")

			if err != nil {
				term.OutputErrorAndExit("Error prompting upgrade trial: %v", err)
			}

			if res {
				err := auth.ConvertTrial()
				if err != nil {
					term.OutputErrorAndExit("Error converting trial: %v", err)
				}
			}

			return
		}

		term.OutputErrorAndExit("Error creating plan: %v", apiErr.Msg)
	}

	err := lib.WriteCurrentPlan(res.Id)

	if err != nil {
		term.OutputErrorAndExit("Error setting current plan: %v", err)
	}

	if name == "" {
		name = "draft"
	}

	fmt.Printf("✅ Started new plan %s and set it to current plan\n", color.New(color.Bold, term.ColorHiGreen).Sprint(name))

	fmt.Println()
	term.PrintCmds("", "load", "tell", "plans", "current")

}
