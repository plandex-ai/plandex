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

var customProvidersOnly bool

var providersCmd = &cobra.Command{
	Use:   "providers",
	Short: "List built-in and custom model providers",
	Run:   listProviders,
}

var showProviderCmd = &cobra.Command{
	Use:   "show [id|name]",
	Short: "Show a custom model provider",
	Args:  cobra.MaximumNArgs(1),
	Run:   showProvider,
}

var addProviderCmd = &cobra.Command{
	Use:     "add",
	Aliases: []string{"create"},
	Short:   "Add a custom model provider",
	Run:     customModelsNotImplemented,
}

var updateProviderCmd = &cobra.Command{
	Use:     "update",
	Aliases: []string{"edit"},
	Short:   "Update a custom model provider",
	Run:     customModelsNotImplemented,
}

func init() {
	RootCmd.AddCommand(providersCmd)
	providersCmd.Flags().BoolVarP(&customProvidersOnly, "custom", "c", false, "List custom providers only")
	providersCmd.AddCommand(showProviderCmd)
	providersCmd.AddCommand(addProviderCmd)
	providersCmd.AddCommand(updateProviderCmd)
}

func listProviders(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()

	var customProviders []*shared.CustomProvider
	var apiErr *shared.ApiError

	if customProvidersOnly && auth.Current.IsCloud {
		term.OutputErrorAndExit("Custom providers are not supported on Plandex Cloud")
		return
	}

	if !auth.Current.IsCloud {
		term.StartSpinner("")
		customProviders, apiErr = api.Client.ListCustomProviders()
		term.StopSpinner()
		if apiErr != nil {
			term.OutputErrorAndExit("Error fetching providers: %v", apiErr.Msg)
			return
		}
	}

	if customProvidersOnly && len(customProviders) == 0 {
		fmt.Println("🤷‍♂️  No custom providers")
		fmt.Println()
		term.PrintCmds("", "providers add")
		return
	}

	color.New(color.Bold, term.ColorHiCyan).Println("🏠 Built-in Providers")
	table := tablewriter.NewWriter(os.Stdout)
	table.SetAutoWrapText(true)
	table.SetHeader([]string{"ID", "Base URL", "API Key", "Other Vars"})
	for _, p := range shared.AllModelProviders {
		if p == shared.ModelProviderCustom {
			continue
		}
		config := shared.BuiltInModelProviderConfigs[p]
		if config.LocalOnly && auth.Current.IsCloud {
			continue
		}
		var apiKey string
		if config.ApiKeyEnvVar != "" {
			apiKey = config.ApiKeyEnvVar
		} else if config.SkipAuth {
			apiKey = "No Auth"
		}

		extraVars := []string{}
		for _, v := range config.ExtraAuthVars {
			extraVars = append(extraVars, v.Var)
		}

		table.Append([]string{
			string(p),
			config.BaseUrl,
			apiKey,
			strings.Join(extraVars, "\n"),
		})
	}
	table.Render()
	fmt.Println()

	if len(customProviders) > 0 {
		color.New(color.Bold, term.ColorHiCyan).Println("🛠️  Custom Providers")
		table := tablewriter.NewWriter(os.Stdout)
		table.SetAutoWrapText(false)
		table.SetHeader([]string{"#", "Name", "Base URL", "API Key", "Other Vars"})
		for i, p := range customProviders {
			extraVars := []string{}
			for _, v := range p.ExtraAuthVars {
				extraVars = append(extraVars, v.Var)
			}
			apiKey := p.ApiKeyEnvVar
			if apiKey == "" && p.SkipAuth {
				apiKey = "No Auth"
			}

			table.Append([]string{
				strconv.Itoa(i + 1),
				p.Name,
				p.BaseUrl,
				apiKey,
				strings.Join(extraVars, "\n"),
			})
		}
		table.Render()
		fmt.Println()
	}

	term.PrintCmds("", "providers show", "providers update", "models import")
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
