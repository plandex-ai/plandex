package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"plandex/api"
	"plandex/auth"
	"plandex/lib"

	"github.com/plandex/plandex/shared"
	"github.com/spf13/cobra"
)

var contextRmCmd = &cobra.Command{
	Use:     "rm",
	Aliases: []string{"remove"},
	Short:   "Remove context",
	Long:    `Remove context by index, name, or glob.`,
	Args:    cobra.MinimumNArgs(1),
	Run:     contextRm,
}

func contextRm(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()
	lib.MustResolveProject()

	if lib.CurrentPlanId == "" {
		fmt.Fprintln(os.Stderr, "No current plan")
		return
	}

	contexts, err := api.Client.ListContext(lib.CurrentPlanId)

	if err != nil {
		fmt.Fprintln(os.Stderr, "Error retrieving context:", err)
		return
	}

	deleteIds := map[string]bool{}

	for i, context := range contexts {
		for _, id := range args {
			if fmt.Sprintf("%d", i+1) == id || context.Name == id || context.FilePath == id || context.Url == id {
				deleteIds[context.Id] = true
				break
			} else if context.FilePath != "" {
				// Check if id is a glob pattern
				matched, err := filepath.Match(id, context.FilePath)
				if err != nil {
					fmt.Fprintln(os.Stderr, "Error matching glob pattern:", err)
					return
				}
				if matched {
					deleteIds[context.Id] = true
					break
				}
			}
		}
	}

	if len(deleteIds) > 0 {
		res, err := api.Client.DeleteContext(lib.CurrentPlanId, shared.DeleteContextRequest{
			Ids: deleteIds,
		})

		if err != nil {
			fmt.Fprintln(os.Stderr, "Error deleting context:", err)
			return
		}

		fmt.Println("✅ " + res.Msg)
	} else {
		fmt.Println("🤷‍♂️ No context removed")
	}
}

func init() {
	RootCmd.AddCommand(contextRmCmd)
}
