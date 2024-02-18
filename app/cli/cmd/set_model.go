package cmd

import (
	"fmt"
	"plandex/api"
	"plandex/auth"
	"plandex/lib"
	"plandex/term"

	"github.com/plandex/plandex/shared"
	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(modelsSetCmd)
}

var modelsSetCmd = &cobra.Command{
	Use:   "set-model [role-or-setting] [property-or-value] [value]",
	Short: "Update model settings",
	Run:   models,
	Args:  cobra.MaximumNArgs(3),
}

func modelsSet(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()
	lib.MustResolveProject()

	settings, err := api.Client.GetSettings(lib.CurrentPlanId, lib.CurrentBranch)
	if err != nil {
		fmt.Println("Error getting current settings:", err)
		return
	}

	var roleOrSetting, property, value string
	if len(args) == 0 {
		// Prompt user to select between updating a top-level setting or a modelset role
		choices := []string{"maxConvoTokens", "maxTokens", "reserveOutputTokens", "ModelSet Role"}
		selection, err := term.SelectFromList("Select a setting or role to update:", choices)
		if err != nil {
			fmt.Println("Error selecting setting or role:", err)
			return
		}
		if selection == "ModelSet Role" {
			// Prompt for modelset role selection
			roleChoices := []string{"core-planner", "summarizer", "builder", "names", "commit-messages", "auto-complete"}
			roleOrSetting, err = term.SelectFromList("Select a role:", roleChoices)
			if err != nil {
				fmt.Println("Error selecting role:", err)
				return
			}
			// Prompt for property of the modelset
			property, err = term.GetUserStringInput("Enter the property to update (e.g., Model, Max 🪙, Temperature):")
			if err != nil {
				fmt.Println("Error getting property:", err)
				return
			}
		} else {
			roleOrSetting = selection
		}
	} else {
		roleOrSetting = args[0]
	}

	if len(args) > 1 {
		property = args[1]
	}
	if len(args) > 2 {
		value = args[2]
	}

	// Logic to update settings based on the roleOrSetting, property, and value
	if roleOrSetting == "maxConvoTokens" || roleOrSetting == "maxTokens" || roleOrSetting == "reserveOutputTokens" {
		updateTopLevelSetting(&settings, roleOrSetting, value)
	} else {
		updateModelSetProperty(&settings.ModelSet, roleOrSetting, property, value)
	}

	err = api.Client.UpdateSettings(lib.CurrentPlanId, lib.CurrentBranch, shared.UpdateSettingsRequest{
		Settings: settings,
	})
	if err != nil {
		fmt.Println("Error updating settings:", err)
		return
	}

	fmt.Println("Settings updated successfully.")
}
