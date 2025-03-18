package cmd

import (
	"fmt"
	"os"
	"plandex-cli/api"
	"plandex-cli/auth"
	"plandex-cli/lib"
	"plandex-cli/term"
	shared "plandex-shared"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/eiannone/keyboard"
	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/shopspring/decimal"
	"github.com/spf13/cobra"
)

const MaxCreditsLogPageSize = 500

var logCreditsPageSize int
var logCreditsPage int

var logCreditsDebitsOnly bool
var logCreditsCreditsOnly bool

var showUsageLog bool

var creditsSession bool
var creditsToday bool
var creditsMonth bool
var creditsCurrentPlan bool

var usageCmd = &cobra.Command{
	Use:   "usage",
	Short: "Display credits balance and usage report",
	Run:   usage,
}

func init() {
	RootCmd.AddCommand(usageCmd)

	usageCmd.Flags().BoolVar(&showUsageLog, "log", false, "Show usage log")
	usageCmd.Flags().IntVarP(&logCreditsPageSize, "page-size", "s", 100, "Number of transactions to display per page")
	usageCmd.Flags().IntVarP(&logCreditsPage, "page", "p", 1, "Page number to display")
	usageCmd.Flags().BoolVar(&logCreditsDebitsOnly, "debits", false, "Show only debits in the log")
	usageCmd.Flags().BoolVar(&logCreditsCreditsOnly, "purchases", false, "Show only purchases in the log")

	usageCmd.Flags().BoolVar(&creditsToday, "today", false, "Show usage for today")
	usageCmd.Flags().BoolVar(&creditsMonth, "month", false, "Show usage for current billing month")
	usageCmd.Flags().BoolVar(&creditsCurrentPlan, "plan", false, "Show usage for the current plan")
}

func usage(cmd *cobra.Command, args []string) {
	if showUsageLog {
		showLog(cmd, args)
	} else {
		showUsage()
	}
}

func showUsage() {
	auth.MustResolveAuthWithOrg()

	term.StartSpinner("")

	if !(creditsSession || creditsToday || creditsMonth || creditsCurrentPlan) {
		if os.Getenv("PLANDEX_REPL_SESSION_ID") != "" {
			creditsSession = true
		} else {
			creditsToday = true
		}
	}

	var sessionId string
	if creditsSession {
		sessionId = os.Getenv("PLANDEX_REPL_SESSION_ID")
		if sessionId == "" {
			term.OutputErrorAndExit("Session ID is not set. The --session flag should be used in the Plandex REPL.")
		}
	}

	var dayStart *time.Time
	if creditsToday {
		now := time.Now()
		midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		dayStart = &midnight
	}

	var planId string
	var currentPlanName string
	if creditsCurrentPlan {
		lib.MustResolveProject()
		planId = lib.CurrentPlanId
		plan, apiErr := api.Client.GetPlan(planId)
		if apiErr != nil {
			term.OutputErrorAndExit("Error getting plan: %v", apiErr)
		}
		currentPlanName = plan.Name
	}

	req := shared.CreditsLogRequest{
		SessionId: sessionId,
		DayStart:  dayStart,
		Month:     creditsMonth,
		PlanId:    planId,
	}

	res, apiErr := api.Client.GetCreditsSummary(req)
	term.StopSpinner()

	if apiErr != nil {
		term.OutputErrorAndExit("Error getting credits summary: %v", apiErr)
	}

	builder := strings.Builder{}

	balance := res.Balance
	balanceStr := formatSpend(balance)

	spendLbl := "💸 Spent"
	if creditsSession {
		spendLbl += " This Session"
	} else if creditsToday {
		spendLbl += " Today"
	} else if creditsMonth {
		spendLbl += " This Billing Month"
		spendLbl += fmt.Sprintf(" (since %s)", res.MonthStart.Format("Jan 2"))
	} else if creditsCurrentPlan {
		spendLbl += fmt.Sprintf(" On Plan 📋 %s", currentPlanName)
	}

	var spendStr string
	if res.TotalSpend.IsZero() {
		spendStr = "$0.00"
	} else {
		spendStr = formatSpend(res.TotalSpend)
	}

	table := tablewriter.NewWriter(&builder)
	table.SetAutoWrapText(false)
	table.SetHeader([]string{"💰 Current Balance", spendLbl})
	table.Append([]string{balanceStr, spendStr})
	table.Render()
	fmt.Fprintln(&builder)

	if !res.CacheSavings.IsZero() {
		table := tablewriter.NewWriter(&builder)
		table.SetAutoWrapText(false)
		table.SetHeader([]string{"🎯 Cache Savings"})
		table.Append([]string{formatSpend(res.CacheSavings)})
		table.Render()
		fmt.Fprintln(&builder)
	}

	amountByStr := map[string]float64{}
	if len(res.ByPlanId) > 0 {
		if !creditsCurrentPlan {
			table := tablewriter.NewWriter(&builder)
			table.SetAutoWrapText(false)
			table.SetHeader([]string{"📋 Plan", "💸 Spent"})

			rows := [][]string{}

			for id, spend := range res.ByPlanId {
				name := res.PlanNamesById[id]
				spendStr := formatSpend(spend)
				rows = append(rows, []string{name, spendStr})
				amountByStr[spendStr] = spend.InexactFloat64()
			}
			sort.Slice(rows, func(i, j int) bool {
				return amountByStr[rows[i][1]] > amountByStr[rows[j][1]]
			})
			for _, row := range rows {
				table.Append(row)
			}

			table.Render()
			fmt.Fprintln(&builder)
		}
	}

	if len(res.ByPurpose) > 0 {
		table = tablewriter.NewWriter(&builder)
		table.SetAutoWrapText(false)
		table.SetHeader([]string{"⚡️ Purpose", "💸 Spent"})

		rows := [][]string{}
		for name, spend := range res.ByPurpose {
			spendStr := formatSpend(spend)
			rows = append(rows, []string{name, spendStr})
			amountByStr[spendStr] = spend.InexactFloat64()
		}
		sort.Slice(rows, func(i, j int) bool {
			return amountByStr[rows[i][1]] > amountByStr[rows[j][1]]
		})
		for _, row := range rows {
			table.Append(row)
		}

		table.Render()
		fmt.Fprintln(&builder)
	}

	if len(res.ByModelName) > 0 {
		table = tablewriter.NewWriter(&builder)
		table.SetAutoWrapText(false)
		table.SetHeader([]string{"🤖 Model", "💸 Spent"})

		rows := [][]string{}
		for name, spend := range res.ByModelName {
			spendStr := formatSpend(spend)
			rows = append(rows, []string{name, spendStr})
			amountByStr[spendStr] = spend.InexactFloat64()
		}
		sort.Slice(rows, func(i, j int) bool {
			return amountByStr[rows[i][1]] > amountByStr[rows[j][1]]
		})
		for _, row := range rows {
			table.Append(row)
		}
		table.Render()
		fmt.Fprintln(&builder)
	}

	term.PageOutput(builder.String())

	term.PrintCmds("", "usage", "billing")
}

func showLog(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()

	if !(creditsSession || creditsToday || creditsMonth || creditsCurrentPlan) {
		if os.Getenv("PLANDEX_REPL_SESSION_ID") != "" {
			creditsSession = true
		} else {
			creditsToday = true
		}
	}

	term.StartSpinner("")

	var transactionType shared.CreditsTransactionType

	if logCreditsDebitsOnly {
		transactionType = shared.CreditsTransactionTypeDebit
	} else if logCreditsCreditsOnly {
		transactionType = shared.CreditsTransactionTypeCredit
	}

	var sessionId string
	if creditsSession {
		sessionId = os.Getenv("PLANDEX_REPL_SESSION_ID")
		if sessionId == "" {
			term.OutputErrorAndExit("Session ID is not set. The --session flag should be used in the Plandex REPL.")
		}
	}

	var dayStart *time.Time
	if creditsToday {
		now := time.Now()
		midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		dayStart = &midnight
	}

	var planId string
	var planName string
	if creditsCurrentPlan {
		lib.MustResolveProject()
		planId = lib.CurrentPlanId

		plan, err := api.Client.GetPlan(planId)
		if err != nil {
			term.OutputErrorAndExit("Error getting plan: %v", err)
			return
		}
		planName = plan.Name
	}

	req := shared.CreditsLogRequest{
		TransactionType: transactionType,
		SessionId:       sessionId,
		DayStart:        dayStart,
		Month:           creditsMonth,
		PlanId:          planId,
	}

	res, apiErr := api.Client.GetCreditsTransactions(logCreditsPageSize, logCreditsPage, req)
	term.StopSpinner()

	if apiErr != nil {
		term.OutputErrorAndExit("Error getting credits transactions: %v", apiErr)
		return
	}

	transactions := res.Transactions

	if len(transactions) == 0 {
		lbl := "🤷‍♂️ No usage"
		if sessionId != "" {
			lbl = "🤷‍♂️ No usage so far this session"
		} else if creditsToday {
			tz, _ := time.Now().Zone()
			lbl = fmt.Sprintf("🤷‍♂️ No usage so far today (since midnight %s)", tz)
		} else if creditsMonth {
			lbl = fmt.Sprintf("🤷‍♂️ No usage so far this billing month (since %s)", res.MonthStart.Format("Jan 2"))
		} else if creditsCurrentPlan {
			lbl = "🤷‍♂️ No usage so far for current plan 👉 " + planName
		}
		fmt.Println(lbl)
		return
	}

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

			if transaction.DebitPlanId != nil {
				planName := res.PlanNamesById[*transaction.DebitPlanId]
				desc += fmt.Sprintf("Plan → %s\n", planName)
			}

			surchargePct := transaction.DebitSurcharge.Div(*transaction.DebitBaseAmount)

			inputPrice := transaction.DebitModelInputPricePerToken.Mul(decimal.NewFromInt(1000000)).Mul(surchargePct.Add(decimal.NewFromInt(1))).StringFixed(4)
			outputPrice := transaction.DebitModelOutputPricePerToken.Mul(decimal.NewFromInt(1000000)).Mul(surchargePct.Add(decimal.NewFromInt(1))).StringFixed(4)

			var cacheDiscountStr string
			var cacheDiscountPct float64
			if transaction.DebitCacheDiscount != nil {
				cacheDiscountStr = transaction.DebitCacheDiscount.StringFixed(4)
				totalAmount := transaction.DebitBaseAmount.Add(*transaction.DebitCacheDiscount)
				cacheDiscountPct = transaction.DebitCacheDiscount.Div(totalAmount).Mul(decimal.NewFromInt(100)).InexactFloat64()
			}

			for i := 0; i < 2; i++ {
				inputPrice = strings.TrimSuffix(inputPrice, "0")
				outputPrice = strings.TrimSuffix(outputPrice, "0")
				cacheDiscountStr = strings.TrimSuffix(cacheDiscountStr, "0")
			}

			desc += fmt.Sprintf("⚡️ %s\n", *transaction.DebitPurpose)
			desc += fmt.Sprintf("🧠 %s\n", transaction.ModelString())
			desc += fmt.Sprintf("💳 Price → $%s input / $%s output per 1M\n", inputPrice, outputPrice)
			desc += fmt.Sprintf("🪙 Used → %d input / %d output\n", *transaction.DebitInputTokens, *transaction.DebitOutputTokens)

			if cacheDiscountStr != "" {
				desc += fmt.Sprintf("🎯 Cache discount → $%s (%d%%)\n", cacheDiscountStr, int(cacheDiscountPct))
			}

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
						showLog(cmd, args) // Re-run the log command with the new page
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
				showLog(cmd, args)
			} else {
				fmt.Println()
				fmt.Println("Already on last page.")
				inputFn()
			}
		case 'p':
			if logCreditsPage > 1 {
				logCreditsPage--
				showLog(cmd, args)
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

func formatSpend(spend decimal.Decimal) string {
	if spend.IsZero() {
		return "$0.00"
	}

	spendStr := fmt.Sprintf("$%s", spend.StringFixed(4))
	for i := 0; i < 2; i++ {
		if strings.HasSuffix(spendStr, "0") {
			spendStr = spendStr[:len(spendStr)-1]
		}
	}
	if spendStr == "$0.00" {
		return "<$0.0001"
	}
	return spendStr
}
