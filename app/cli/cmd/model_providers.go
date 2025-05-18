package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"plandex-cli/api"
	"plandex-cli/auth"
	"plandex-cli/term"

	shared "plandex-shared"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

var providersCmd = &cobra.Command{
	Use:   "providers",
	Short: "Manage custom model providers",
}

var listProvidersCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List custom model providers",
	Run:     listProviders,
}

var addProviderCmd = &cobra.Command{
	Use:     "add",
	Aliases: []string{"create"},
	Short:   "Add a custom model provider",
	Run:     addProvider,
}

var showProviderCmd = &cobra.Command{
	Use:   "show [id|name]",
	Short: "Show a custom model provider",
	Args:  cobra.MaximumNArgs(1),
	Run:   showProvider,
}

var updateProviderCmd = &cobra.Command{
	Use:     "update [id|name]",
	Aliases: []string{"edit"},
	Short:   "Update a custom model provider",
	Args:    cobra.MaximumNArgs(1),
	Run:     updateProvider,
}

func init() {
	RootCmd.AddCommand(providersCmd)
	providersCmd.AddCommand(listProvidersCmd)
	providersCmd.AddCommand(addProviderCmd)
	providersCmd.AddCommand(showProviderCmd)
	providersCmd.AddCommand(updateProviderCmd)
}

func listProviders(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()

	term.StartSpinner("")
	providers, apiErr := api.Client.ListCustomProviders()
	term.StopSpinner()
	if apiErr != nil {
		term.OutputErrorAndExit("Error fetching providers: %v", apiErr.Msg)
		return
	}

	if len(providers) == 0 {
		fmt.Println("🤷‍♂️  No custom providers")
		fmt.Println()
		term.PrintCmds("", "providers add")
		return
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"#", "Name", "Base URL", "Skip Auth", "API Key Env Var"})
	for i, p := range providers {
		table.Append([]string{
			strconv.Itoa(i + 1),
			p.Name,
			p.BaseUrl,
			fmt.Sprintf("%v", p.SkipAuth),
			p.ApiKeyEnvVar,
		})
	}
	table.Render()
	fmt.Println()
	term.PrintCmds("", "providers show", "providers update", "providers add")
}

func addProvider(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()

	name, err := term.GetRequiredUserStringInput("Provider name:")
	if err != nil {
		term.OutputErrorAndExit("Error reading name: %v", err)
		return
	}

	baseUrl, err := term.GetRequiredUserStringInput("Base URL:")
	if err != nil {
		term.OutputErrorAndExit("Error reading base URL: %v", err)
		return
	}
	baseUrl = strings.TrimSuffix(baseUrl, "/")

	skipAuth, err := term.ConfirmYesNo("Skip auth?")
	if err != nil {
		term.OutputErrorAndExit("Error reading skip auth: %v", err)
		return
	}

	var apiKeyEnvVar string
	if !skipAuth {
		apiKeyEnvVar, err = term.GetRequiredUserStringInput("API key env var:")
		if err != nil {
			term.OutputErrorAndExit("Error reading env var: %v", err)
			return
		}
	}

	provider := &shared.CustomProvider{
		Name:         name,
		BaseUrl:      baseUrl,
		SkipAuth:     skipAuth,
		ApiKeyEnvVar: apiKeyEnvVar,
	}

	term.StartSpinner("")
	apiErr := api.Client.CreateCustomProvider(provider)
	term.StopSpinner()
	if apiErr != nil {
		term.OutputErrorAndExit("Error creating provider: %v", apiErr.Msg)
		return
	}

	fmt.Println("✅ Added custom provider", color.New(color.Bold, term.ColorHiCyan).Sprint(name))
}

func showProvider(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()

	term.StartSpinner("")
	providers, apiErr := api.Client.ListCustomProviders()
	term.StopSpinner()
	if apiErr != nil {
		term.OutputErrorAndExit("Error fetching providers: %v", apiErr.Msg)
		return
	}

	if len(providers) == 0 {
		fmt.Println("🤷‍♂️  No custom providers")
		fmt.Println()
		term.PrintCmds("", "providers add")
		return
	}

	var selected *shared.CustomProvider
	if len(args) == 1 {
		input := args[0]
		idx, err := strconv.Atoi(input)
		if err == nil && idx > 0 && idx <= len(providers) {
			selected = providers[idx-1]
		} else {
			for _, p := range providers {
				if p.Name == input || p.Id == input {
					selected = p
					break
				}
			}
		}
	}

	if selected == nil {
		opts := make([]string, len(providers))
		for i, p := range providers {
			opts[i] = fmt.Sprintf("%s (%s)", p.Name, p.BaseUrl)
		}
		choice, err := term.SelectFromList("Select provider:", opts)
		if err != nil {
			term.OutputErrorAndExit("Error selecting provider: %v", err)
			return
		}
		for i, o := range opts {
			if o == choice {
				selected = providers[i]
				break
			}
		}
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetAutoWrapText(false)
	table.SetHeader([]string{"Field", "Value"})
	table.Append([]string{"Name", selected.Name})
	table.Append([]string{"Base URL", selected.BaseUrl})
	table.Append([]string{"Skip Auth", fmt.Sprintf("%v", selected.SkipAuth)})
	table.Append([]string{"API Key Env Var", selected.ApiKeyEnvVar})
	if len(selected.ExtraAuthVars) > 0 {
		for i, v := range selected.ExtraAuthVars {
			label := fmt.Sprintf("Extra Auth %d", i+1)
			table.Append([]string{label, v.Var})
		}
	}
	table.Render()
	fmt.Println()
	term.PrintCmds("", "providers update", "providers list")
}

func updateProvider(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()

	term.StartSpinner("")
	providers, apiErr := api.Client.ListCustomProviders()
	term.StopSpinner()
	if apiErr != nil {
		term.OutputErrorAndExit("Error fetching providers: %v", apiErr.Msg)
		return
	}

	if len(providers) == 0 {
		fmt.Println("🤷‍♂️  No custom providers")
		fmt.Println()
		term.PrintCmds("", "providers add")
		return
	}

	var selected *shared.CustomProvider
	if len(args) == 1 {
		input := args[0]
		idx, err := strconv.Atoi(input)
		if err == nil && idx > 0 && idx <= len(providers) {
			selected = providers[idx-1]
		} else {
			for _, p := range providers {
				if p.Name == input || p.Id == input {
					selected = p
					break
				}
			}
		}
	}

	if selected == nil {
		opts := make([]string, len(providers))
		for i, p := range providers {
			opts[i] = fmt.Sprintf("%s (%s)", p.Name, p.BaseUrl)
		}
		choice, err := term.SelectFromList("Select provider:", opts)
		if err != nil {
			term.OutputErrorAndExit("Error selecting provider: %v", err)
			return
		}
		for i, o := range opts {
			if o == choice {
				selected = providers[i]
				break
			}
		}
	}

	newName, err := term.GetRequiredUserStringInputWithDefault("Provider name:", selected.Name)
	if err != nil {
		term.OutputErrorAndExit("Error reading name: %v", err)
		return
	}

	newBaseUrl, err := term.GetRequiredUserStringInputWithDefault("Base URL:", selected.BaseUrl)
	if err != nil {
		term.OutputErrorAndExit("Error reading base URL: %v", err)
		return
	}
	newBaseUrl = strings.TrimSuffix(newBaseUrl, "/")

	newSkipAuth := selected.SkipAuth
	changeSkip, err := term.ConfirmYesNo("Change skip auth? (current: " + fmt.Sprintf("%v", selected.SkipAuth) + ")")
	if err == nil && changeSkip {
		newSkipAuth, err = term.ConfirmYesNo("Skip auth?")
		if err != nil {
			term.OutputErrorAndExit("Error reading skip auth: %v", err)
			return
		}
	}

	var newApiKeyEnvVar string
	if !newSkipAuth {
		newApiKeyEnvVar, err = term.GetRequiredUserStringInputWithDefault("API key env var:", selected.ApiKeyEnvVar)
		if err != nil {
			term.OutputErrorAndExit("Error reading env var: %v", err)
			return
		}
	}

	provider := &shared.CustomProvider{
		Id:            selected.Id,
		Name:          newName,
		BaseUrl:       newBaseUrl,
		SkipAuth:      newSkipAuth,
		ApiKeyEnvVar:  newApiKeyEnvVar,
		ExtraAuthVars: selected.ExtraAuthVars,
	}

	term.StartSpinner("")
	apiErr = api.Client.UpdateCustomProvider(provider)
	term.StopSpinner()
	if apiErr != nil {
		term.OutputErrorAndExit("Error updating provider: %v", apiErr.Msg)
		return
	}

	fmt.Println("✅ Updated custom provider", color.New(color.Bold, term.ColorHiCyan).Sprint(newName))
}
