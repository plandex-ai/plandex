package cmd

import (
	"fmt"
	"path/filepath"
	"plandex/api"
	"plandex/auth"
	"plandex/lib"
	"plandex/term"
	"strconv"
	"strings"

	"github.com/plandex/plandex/shared"
	"github.com/spf13/cobra"
)

var contextRmCmd = &cobra.Command{
	Use:     "rm",
	Aliases: []string{"remove", "unload"},
	Short:   "Remove context",
	Long:    `Remove context by index, name, or glob.`,
	Args:    cobra.MinimumNArgs(1),
	Run:     contextRm,
}

func parseIndices(args []string) map[int]bool {
	indices := map[int]bool{}
	for _, arg := range args {
		if strings.Contains(arg, "-") {
			parts := strings.Split(arg, "-")
			start, err1 := strconv.Atoi(parts[0])
			end, err2 := strconv.Atoi(parts[1])
			if err1 == nil && err2 == nil && start <= end {
				for i := start; i <= end; i++ {
					indices[i] = true
				}
			}
		} else {
			index, err := strconv.Atoi(arg)
			if err == nil {
				indices[index] = true
			}
		}
	}
	return indices
}

func contextRm(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()
	lib.MustResolveProject()

	if lib.CurrentPlanId == "" {
		term.OutputNoCurrentPlanErrorAndExit()
	}

	term.StartSpinner("")
	contexts, err := api.Client.ListContext(lib.CurrentPlanId, lib.CurrentBranch)

	if err != nil {
		term.OutputErrorAndExit("Error retrieving context: %v", err)
	}

	deleteIds := map[string]bool{}
	indices := parseIndices(args)

	for i, context := range contexts {
		if indices[i+1] {
			deleteIds[context.Id] = true
			continue
		}
		for _, id := range args {
			if context.Name == id || context.FilePath == id || context.Url == id {
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


