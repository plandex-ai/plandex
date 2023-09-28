package cmd

import (
	"fmt"
	"os"

	"plandex/lib"

	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(lsCmd)
}

// lsCmd represents the list command
var lsCmd = &cobra.Command{
	Use:     "ls",
	Aliases: []string{"list"},
	Short:   "List all available plans",
	Run:     ls,
}

func ls(cmd *cobra.Command, args []string) {
	plandexDir, _, err := lib.FindOrCreatePlandex()

	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		return
	}

	plans, err := os.ReadDir(plandexDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		return
	}

	fmt.Println("Available plans:")
	for _, p := range plans {
		if p.IsDir() {
			fmt.Println("-", p.Name())
		}
	}
}
