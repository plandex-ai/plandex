package cmd

import (
	"fmt"
	"os"
	"plandex/api"
	"plandex/auth"
	"plandex/lib"

	"github.com/plandex/plandex/shared"
	"github.com/spf13/cobra"
)

var clearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear all context",
	Long:  `Clear all context.`,
	Run:   clearAllContext,
}

func clearAllContext(cmd *cobra.Command, args []string) {
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

	for _, context := range contexts {
		deleteIds[context.Id] = true
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
	RootCmd.AddCommand(clearCmd)
}
