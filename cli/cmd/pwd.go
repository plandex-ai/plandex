package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"plandex/lib"

	"github.com/spf13/cobra"
)

var pwdCmd = &cobra.Command{
	Use:   "pwd",
	Short: "Get the current plan directory.",
	Long:  "Use it like this `cd $(plandex pwd)` to go to the current plan directory.",
	Run:   pwd,
}

func init() {
	RootCmd.AddCommand(pwdCmd)
}

func pwd(cmd *cobra.Command, args []string) {
	if lib.CurrentPlanName == "" {
		fmt.Fprintf(os.Stderr, "No current plan.\n")
		os.Exit(1)
	}
	fmt.Println(filepath.Join(lib.PlandexDir, lib.CurrentPlanName))

}
