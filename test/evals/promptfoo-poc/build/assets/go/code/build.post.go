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

func parseRange(arg string) ([]int, error) {
	var indices []int
	parts := strings.Split(arg, "-")
	if len(parts) == 2 {
		start, err := strconv.Atoi(parts[0])
		if err != nil {
			return nil, err
		}
		end, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, err
		}
		for i := start; i <= end; i++ {
			indices = append(indices, i)
		}
	} else {
		index, err := strconv.Atoi(arg)
		if err != nil {
			return nil, err
		}
		indices = append(indices, index)
	}
	return indices, nil
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

	for _, arg := range args {
		indices, err := parseRange(arg)
		if err != nil {
			term.OutputErrorAndExit("Error parsing range: %v", err)
		}

		for _, index := range indices {
			if index > 0 && index <= len(contexts) {
				context := contexts[index-1]
				deleteIds[context.Id] = true
			}
		}
	}

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
