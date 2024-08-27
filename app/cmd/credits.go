package cmd

import (
	"fmt"
	"os"
	"plandex/api"
	"plandex/auth"
	"plandex/term"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

var creditsCmd = &cobra.Command{
	Use:   "credits",
	Short: "Display the current credit balance if using integrated models mode",
	Run:   credits,
}

func init() {
	RootCmd.AddCommand(creditsCmd)
}

func credits(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()

	org, apiErr := api.Client.GetOrgSession()
	if apiErr != nil {
		term.OutputErrorAndExit("Error getting org session: %v", apiErr)
		return
	}

	if !org.IntegratedModelsMode {
		fmt.Println("The org isn't using integrated models mode.")
		return
	}

	balance := org.CloudBillingFields.CreditsBalance
	balanceStr := fmt.Sprintf("$%.4f", balance)

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Current Credit Balance"})
	table.Append([]string{balanceStr})
	table.Render()
}
