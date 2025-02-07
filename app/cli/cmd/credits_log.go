package cmd

import (
	"fmt"
	"plandex-cli/api"
	"plandex-cli/auth"
	"plandex-cli/term"
	"strconv"
	"strings"
	"unicode"

	shared "plandex-shared"

	"github.com/eiannone/keyboard"
	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/shopspring/decimal"
	"github.com/spf13/cobra"
)

const MaxCreditsLogPageSize = 500

var logCreditsPageSize int
var logCreditsPage int

var usageCmd = &cobra.Command{
	Use:   "usage",
	Short: "Display the credits transactions log",
	Run:   creditsLog,
}

func init() {
	usageCmd.Flags().IntVarP(&logCreditsPageSize, "page-size", "s", 100, "Number of transactions to display per page")
	usageCmd.Flags().IntVarP(&logCreditsPage, "page", "p", 1, "Page number to display")

	RootCmd.AddCommand(usageCmd)
}

func creditsLog(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()

	term.StartSpinner("")
	res, apiErr := api.Client.GetCreditsTransactions(logCreditsPageSize, logCreditsPage)
	term.StopSpinner()

	if apiErr != nil {
		term.OutputErrorAndExit("Error getting credits transactions: %v", apiErr)
		return
	}

	transactions := res.Transactions

	tableString := &strings.Builder{}
	table := tablewriter.NewWriter(tableString)
	table.SetAutoWrapText(false)
	table.SetHeader([]string{"Amount", "Balance", "Transaction"})

	for _, transaction := range transactions {
		var sign string
		var c color.Attribute

		desc := transaction.CreatedAt.Local().Format("2006-01-02 15:04:05.000 EST") + "\n"

		if transaction.TransactionType == "debit" {
			sign = "-"
			c = term.ColorHiRed

			if transaction.DebitPlanName != nil {
				desc += fmt.Sprintf("Plan → %s\n", *transaction.DebitPlanName)
			}

			surchargePct := transaction.DebitSurcharge.Div(*transaction.DebitBaseAmount)

			inputPrice := transaction.DebitModelInputPricePerToken.Mul(decimal.NewFromInt(1000000)).Mul(surchargePct.Add(decimal.NewFromInt(1))).StringFixed(4)
			outputPrice := transaction.DebitModelOutputPricePerToken.Mul(decimal.NewFromInt(1000000)).Mul(surchargePct.Add(decimal.NewFromInt(1))).StringFixed(4)

			for i := 0; i < 2; i++ {
				inputPrice = strings.TrimSuffix(inputPrice, "0")
				outputPrice = strings.TrimSuffix(outputPrice, "0")
			}

			desc += fmt.Sprintf("⚡️ %s\n🧠 %s/%s\n💳 Price → $%s input / $%s output per 1M\n🪙 Used → %d input / %d output\n", *transaction.DebitPurpose, string(*transaction.DebitModelProvider), *transaction.DebitModelName, inputPrice, outputPrice, *transaction.DebitInputTokens, *transaction.DebitOutputTokens)

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
			case shared.CreditTypeSwitch:
				desc += "Switched to Integrated Models mode"
			}

			desc += "\n"
		}

		amountStr := transaction.Amount.StringFixed(6)
		for i := 0; i < 4; i++ {
			amountStr = strings.TrimSuffix(amountStr, "0")
		}

		balanceStr := transaction.EndBalance.StringFixed(4)
		for i := 0; i < 2; i++ {
			balanceStr = strings.TrimSuffix(balanceStr, "0")
		}

		table.Append([]string{
			color.New(c).Sprint(sign + "$" + amountStr),
			"$" + balanceStr,

			desc,
		})
	}

	table.Render()

	var output string
	var pageLine string

	if res.NumPages > 1 {
		pageLine = fmt.Sprintf("Page size %d. Showing page %d of %d", logCreditsPageSize, logCreditsPage, res.NumPages)
		if res.NumPagesMax {
			pageLine = "+"
		}
		output = pageLine + "\n\n" + tableString.String()
	} else {
		output = tableString.String()
	}

	term.PageOutput(output)

	var inputFn func()
	inputFn = func() {
		fmt.Println("\n" + pageLine)

		prompts := []string{}

		if res.NumPages > 1 && logCreditsPage < res.NumPages {
			prompts = append(prompts, "Press 'n' for next page")
		}

		if logCreditsPage > 1 {
			prompts = append(prompts, "Press 'p' for previous page")
		}

		prompts = append(prompts, "Type any number and press enter to jump to a page")

		prompts = append(prompts, "Press 'q' to quit")

		color.New(term.ColorHiMagenta, color.Bold).Println(strings.Join(prompts, "\n"))
		color.New(term.ColorHiMagenta, color.Bold).Print("> ")

		char, _, err := term.GetUserKeyInput()

		if err != nil {
			term.OutputErrorAndExit("Failed to get user input: %v", err)
		}

		// Check if the input is a digit
		if unicode.IsDigit(char) {
			var numberInput strings.Builder
			numberInput.WriteRune(char)

			fmt.Print(string(char)) // Show the initial digit

			for {
				char, key, err := term.GetUserKeyInput()
				if err != nil {
					term.OutputErrorAndExit("Failed to get user input: %v", err)
				}

				// If Enter is pressed, commit the input
				if key == keyboard.KeyEnter {
					pageNumber, err := strconv.Atoi(numberInput.String())
					if err != nil {
						fmt.Println("Invalid page number.")
						return
					}

					// Check if the page number is valid
					if pageNumber >= 1 && (pageNumber <= res.NumPages || res.NumPagesMax) {
						logCreditsPage = pageNumber
						creditsLog(cmd, args) // Re-run the log command with the new page
					} else {
						fmt.Println()
						fmt.Println("Invalid page number.")
						inputFn()
					}
					return
				}

				// If another digit is pressed, add it to the input
				if unicode.IsDigit(char) {
					numberInput.WriteRune(char)
					fmt.Print(string(char)) // Show the digit
				} else if key == keyboard.KeyBackspace || key == keyboard.KeyBackspace2 {
					// Handle backspace
					if numberInput.Len() > 0 {
						// Remove the last rune
						input := numberInput.String()
						numberInput.Reset()
						numberInput.WriteString(input[:len(input)-1])
						fmt.Print("\b \b") // Erase the digit
					}

				} else {
					// Handle invalid input while typing a number
					fmt.Println()
					fmt.Println("\nInvalid input. Please enter a valid page number.")
					inputFn()
					return
				}
			}
		}

		// Handle non-digit hotkeys
		fmt.Print(string(char))
		switch char {
		case 'n':
			if logCreditsPage < res.NumPages || res.NumPagesMax {
				logCreditsPage++
				creditsLog(cmd, args)
			} else {
				fmt.Println()
				fmt.Println("Already on last page.")
				inputFn()
			}
		case 'p':
			if logCreditsPage > 1 {
				logCreditsPage--
				creditsLog(cmd, args)
			} else {
				fmt.Println()
				fmt.Println("Already on first page.")
				inputFn()
			}
		case 'q':
			fmt.Println()
			return
		default:
			fmt.Println()
			fmt.Println("Invalid input.")
			inputFn()
		}

	}

	if res.NumPages > 1 {
		inputFn()
	}
}
