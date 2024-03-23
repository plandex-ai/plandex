package cmd

import (
	"fmt"
	"path/filepath"
	"plandex/api"
	"plandex/auth"
	"plandex/lib"
	"plandex/term"

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
		fmt.Println("🤷‍♂️ No current plan")
		return
	}

	term.StartSpinner("")
	contexts, err := api.Client.ListContext(lib.CurrentPlanId, lib.CurrentBranch)

	if err != nil {
		term.OutputErrorAndExit("Error retrieving context: %v", err)
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
					term.OutputErrorAndExit("Error matching glob pattern: %v", err)
				}
				if matched {
					deleteIds[context.Id] = true
					break
				}

				// Check if id is a parent directory
				parentDir := context.FilePath
				for parentDir != "." && parentDir != "/" && parentDir != "" {
					if parentDir == id {
						deleteIds[context.Id] = true
						break
					}
					parentDir = filepath.Dir(parentDir) // Move up one directory
				}

			}
		}
	}

	if len(deleteIds) > 0 {
		res, err := api.Client.DeleteContext(lib.CurrentPlanId, lib.CurrentBranch, shared.DeleteContextRequest{
			Ids: deleteIds,
		})
		term.StopSpinner()

		if err != nil {
			term.OutputErrorAndExit("Error deleting context: %v", err)
		}

		fmt.Println("✅ " + res.Msg)
	} else {
		term.StopSpinner()
		fmt.Println("🤷‍♂️ No context removed")
	}
}

func init() {
	RootCmd.AddCommand(contextRmCmd)
}
