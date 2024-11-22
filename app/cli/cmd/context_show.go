package cmd

import (
	"fmt"
	"log"
	"plandex/api"
	"plandex/auth"
	"plandex/lib"
	"strconv"

	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(contextShowCmd)
}

var contextShowCmd = &cobra.Command{
	Use:   "show [name-or-index]",
	Short: "Show the body of a context by name or list index",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		auth.MustResolveAuthWithOrg()
		lib.MustResolveProject()

		nameOrIndex := args[0]

		// Get list of contexts first
		contexts, err := api.Client.ListContext(lib.CurrentPlanId, lib.CurrentBranch)
		if err != nil {
			log.Printf("Error listing contexts: %v\n", err)
			return fmt.Errorf("error listing contexts: %v", err)
		}

		var contextId string

		// Try parsing as index first
		if idx, err := strconv.Atoi(nameOrIndex); err == nil {
			// Convert to 0-based index
			idx--
			if idx < 0 || idx >= len(contexts) {
				return fmt.Errorf("invalid context index: %s", nameOrIndex)
			}
			contextId = contexts[idx].Id
		} else {
			// Try finding by name
			found := false
			for _, ctx := range contexts {
				if ctx.Name == nameOrIndex || ctx.FilePath == nameOrIndex {
					contextId = ctx.Id
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("no context found with name: %s", nameOrIndex)
			}
		}

		res, apiErr := api.Client.GetContextBody(lib.CurrentPlanId, lib.CurrentBranch, contextId)
		if apiErr != nil {
			log.Printf("Error getting context body: %v\n", apiErr)
			return fmt.Errorf("error getting context body: %v", apiErr)
		}

		fmt.Println(res.Body)
		return nil
	},
}
