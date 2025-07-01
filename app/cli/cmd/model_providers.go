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
		term.PrintCmds("", "models custom")
		return
	}

	color.New(color.Bold, term.ColorHiCyan).Println("🏠 Built-in Providers")
	table := tablewriter.NewWriter(os.Stdout)
	table.SetAutoWrapText(true)

	var header []string
	if auth.Current.IsCloud {
		header = []string{"ID", "API Key", "Other Vars"}
	} else {
		header = []string{"ID", "Base URL", "API Key", "Other Vars"}
	}
	table.SetHeader(header)
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
		if config.Provider == shared.ModelProviderAmazonBedrock {
			extraVars = append(extraVars, "PLANDEX_AWS_PROFILE")
		}
		for _, v := range config.ExtraAuthVars {
			extraVars = append(extraVars, v.Var)
		}

		if auth.Current.IsCloud {
			table.Append([]string{
				string(p),
				apiKey,
				strings.Join(extraVars, "\n"),
			})
		} else {
			table.Append([]string{
				string(p),
				config.BaseUrl,
				apiKey,
				strings.Join(extraVars, "\n"),
			})
		}
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

	fmt.Println(color.New(color.Bold, term.ColorHiCyan).Sprint("\n📖 Per-provider instructions"))
	fmt.Println("Go to → " + color.New(color.Bold).Sprint("https://docs.plandex.ai/models/model-providers"))
	fmt.Println()

	term.PrintCmds("", "models custom")
}
