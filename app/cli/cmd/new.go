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
	Short:   "Start a new plan.",
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
	lib.MustResolveProject()

	res, apiErr := api.Client.CreatePlan(lib.CurrentProjectId, shared.CreatePlanRequest{Name: name})

	if apiErr != nil {
		if apiErr.Type == shared.ApiErrorTypeTrialPlansExceeded {
			fmt.Fprintf(os.Stderr, "🚨 You've reached the free trial limit of %d plans\n", apiErr.TrialPlansExceededError.MaxPlans)

			res, err := term.ConfirmYesNo("Upgrade trial now?")

			if err != nil {
				fmt.Fprintln(os.Stderr, "Error prompting upgrade trial:", err)
				return
			}

			if res {
				err := auth.ConvertTrial()
				if err != nil {
					fmt.Fprintln(os.Stderr, "Error converting trial:", err)
					return
				}
			}

			return
		}

		fmt.Fprintln(os.Stderr, "Error creating plan:", apiErr.Msg)
		return
	}

	err := lib.SetCurrentPlan(res.Id)

	if err != nil {
		fmt.Fprintln(os.Stderr, "Error setting current plan:", err)
		return
	}

	if name == "" {
		fmt.Println("✅ Started new plan")
	} else {
		fmt.Printf("✅ Started new plan: %s\n", color.New(color.Bold, color.FgHiWhite).Sprint(name))
	}

	fmt.Println()
	term.PrintCmds("", "load", "tell", "plans")

}
