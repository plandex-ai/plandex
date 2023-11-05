package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"plandex/lib"
	"strconv"

	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(cdCmd)
}

var cdCmd = &cobra.Command{
	Use:     "cd [name-or-index]",
	Aliases: []string{"set-plan"},
	Short:   "Change to a different plan by name or index",
	Args:    cobra.ExactArgs(1),
	Run:     cd,
}

func cd(cmd *cobra.Command, args []string) {
	plandexDir, _, err := lib.FindOrCreatePlandex()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		return
	}

	nameOrIdx := args[0]
	var name string

	// see if it's an index
	if idx, err := strconv.Atoi(nameOrIdx); err == nil {
		plans, err := lib.GetPlans()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error getting plans:", err)
		}
		plan := plans[idx]
		name = plan.Name
	} else {
		name = nameOrIdx
	}

	planDir := filepath.Join(plandexDir, name)

	if _, err := os.Stat(planDir); os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, "🤷‍♂️ Plan '"+name+"' does not exist")
		os.Exit(1)
	}

	err = lib.SetCurrentPlan(name)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error setting current plan:", err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stderr, "✅ Changed current plan to "+name)
}
