package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

func init() {
	contextCmd.AddCommand(contextShowCmd)
}

var contextShowCmd = &cobra.Command{
	Use:   "show [context-id]",
	Short: "Show the body of a context",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		contextId := args[0]

		planId, branch, err := getCurrentPlanAndBranch()
		if err != nil {
			return err
		}

		res, apiErr := api.GetContextBody(planId, branch, contextId)
		if apiErr != nil {
			log.Printf("Error getting context body: %v\n", apiErr)
			return fmt.Errorf("error getting context body: %v", apiErr)
		}

		fmt.Println(res.Body)
		return nil
	},
}
