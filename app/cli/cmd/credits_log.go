package cmd

import (
	"fmt"
	"plandex/api"
	"plandex/auth"
	"plandex/term"
	"strings"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/plandex/plandex/shared"
	"github.com/shopspring/decimal"
	"github.com/spf13/cobra"
)

var creditsLogCmd = &cobra.Command{
	Use:   "log",
	Short: "Display the credits transactions log",
	Run:   creditsLog,
}

func init() {
	// subcommand of 'credits'
	creditsCmd.AddCommand(creditsLogCmd)
}

func creditsLog(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()

	term.StartSpinner("")
	transactions, apiErr := api.Client.GetCreditsTransactions()
	term.StopSpinner()

	if apiErr != nil {
		term.OutputErrorAndExit("Error getting credits transactions: %v", apiErr)
		return
	}

	tableString := &strings.Builder{}
	table := tablewriter.NewWriter(tableString)
	table.SetAutoWrapText(false)
	table.SetHeader([]string{"Amount", "Balance", "Transaction"})

	for _, transaction := range transactions {
		var sign string
		var c color.Attribute

		desc := transaction.CreatedAt.Local().Format("2006-01-02 15:04:05.000 PDT") + "\n"

		if transaction.TransactionType == "debit" {
			sign = "-"
			c = term.ColorHiRed

			var t string
			if *transaction.DebitType == shared.DebitTypeModelInput {
				t = "input"
			} else if *transaction.DebitType == shared.DebitTypeModelOutput {
				t = "output"
			}

			if transaction.DebitPlanName != nil {
				desc += fmt.Sprintf("Plan → %s\n", *transaction.DebitPlanName)
			}

			surchargePct := transaction.DebitSurcharge.Div(*transaction.DebitBaseAmount)

			price := transaction.DebitModelPricePerToken.Mul(decimal.NewFromInt(1000000)).Mul(surchargePct.Add(decimal.NewFromInt(1))).StringFixed(4)

			for i := 0; i < 2; i++ {
				if strings.HasSuffix(price, "0") {
					price = price[:len(price)-1]
				}
			}

			desc += fmt.Sprintf("⚡️ %s\n🧠 %s/%s → %s\n💳 Price → $%s per 1M 🪙\n💸 Used → %d 🪙\n", *transaction.DebitPurpose, string(*transaction.DebitModelProvider), *transaction.DebitModelName, t, price, *transaction.DebitTokens)

		} else {
			sign = "+"
			c = term.ColorHiGreen

			switch *transaction.CreditType {
			case shared.CreditTypeGrant:
				desc += "Monthly subscription payment"
			case shared.CreditTypeTrial:
				desc += "Started trial"
			case shared.CreditTypePurchase:
				desc += "Purchased credits"
			}
		}

		amountStr := transaction.Amount.StringFixed(6)
		for i := 0; i < 4; i++ {
			if strings.HasSuffix(amountStr, "0") {
				amountStr = amountStr[:len(amountStr)-1]
			}
		}

		balanceStr := transaction.EndBalance.StringFixed(4)
		for i := 0; i < 2; i++ {
			if strings.HasSuffix(balanceStr, "0") {
				balanceStr = balanceStr[:len(balanceStr)-1]
			}
		}

		table.Append([]string{
			color.New(c).Sprint(sign + "$" + amountStr),
			"$" + balanceStr,

			desc,
		})
	}

	table.Render()

	term.PageOutput(tableString.String())
}
