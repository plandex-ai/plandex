package cmd

import (
	"fmt"
	"os"

	"plandex/api"
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
	lib.MustResolveProject()

	res, err := api.Client.CreatePlan(lib.CurrentProjectId, shared.CreatePlanRequest{Name: name})

	if err != nil {
		fmt.Fprintln(os.Stderr, "Error creating plan:", err)
		return
	}

	err = lib.SetCurrentPlan(res.Id)

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
