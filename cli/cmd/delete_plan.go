package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"plandex/lib"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var all bool

func init() {
	rmCmd.Flags().BoolVar(&all, "all", false, "Delete all plans and clear the current plan")
	RootCmd.AddCommand(rmCmd)
}

// rmCmd represents the rm command
var rmCmd = &cobra.Command{
	Use:     "delete-plan [name]",
	Aliases: []string{"del"},
	Short:   "Delete the specified plan",
	Args:    cobra.RangeArgs(0, 1),
	Run:     del,
}

func del(cmd *cobra.Command, args []string) {
	if all {
		delAll()
		return
	}

	name := args[0]
	plandexDir, _, err := lib.FindOrCreatePlandex()

	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		return
	}

	planDir := filepath.Join(plandexDir, name)

	if _, err := os.Stat(planDir); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Plan %s does not exist", color.New(color.Bold).Sprint(name))
		return
	}

	err = os.RemoveAll(planDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error deleting the plan:", err)
		return
	}

	if lib.CurrentPlanName == name {
		err = os.Remove(filepath.Join(plandexDir, "current_plan.json"))
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error deleting current plan:", err)
			return
		}
	}

	fmt.Printf("✅ Deleted plan %s\n", color.New(color.Bold).Sprint(name))
}

func delAll() {
	plandexDir, _, err := lib.FindOrCreatePlandex()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		return
	}

	err = os.RemoveAll(plandexDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error deleting all the plans:", err)
		return
	}

	err = os.Mkdir(plandexDir, os.ModePerm)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error creating .plandex folder:", err)
		return
	}

	fmt.Fprintln(os.Stderr, "✅ All plans have been deleted.")
}
