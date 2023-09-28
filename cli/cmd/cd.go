package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"plandex/lib"

	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(cdCmd)
}

var cdCmd = &cobra.Command{
	Use:     "cd [name]",
	Aliases: []string{"set-plan"},
	Short:   "Change to a different plan",
	Args:    cobra.ExactArgs(1),
	Run:     cd,
}

func cd(cmd *cobra.Command, args []string) {
	name := args[0]
	plandexDir, _, err := lib.FindOrCreatePlandex()

	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	planDir := filepath.Join(plandexDir, name)

	if _, err := os.Stat(planDir); os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, "Error: plan with name '"+name+"' does not exist")
		os.Exit(1)
	}

	currentPlanFilePath := filepath.Join(plandexDir, "current_plan.json")
	currentPlanFile, err := os.OpenFile(currentPlanFilePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer currentPlanFile.Close()

	_, err = currentPlanFile.WriteString(fmt.Sprintf(`{"name": "%s"}`, name))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stderr, "✅ Changed current plan to "+name)
}
