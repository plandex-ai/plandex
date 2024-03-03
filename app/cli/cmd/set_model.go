package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"plandex/api"
	"plandex/auth"
	"plandex/lib"
	"plandex/term"
	"reflect"
	"strconv"
	"strings"

	"github.com/plandex/plandex/shared"
	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(modelsSetCmd)
}

var modelsSetCmd = &cobra.Command{
	Use:   "set-model [role-or-setting] [property-or-value] [value]",
	Short: "Update model settings",
	Run:   modelsSet,
	Args:  cobra.MaximumNArgs(3),
}

func modelsSet(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()
	lib.MustResolveProject()

	term.StartSpinner("")
	originalSettings, apiErr := api.Client.GetSettings(lib.CurrentPlanId, lib.CurrentBranch)
	term.StopSpinner()

	if apiErr != nil {
		term.OutputErrorAndExit("Error getting current settings: %v", apiErr)
		return
	}

	// Marshal and unmarshal to make a deep copy of the settings
	jsonBytes, err := json.Marshal(originalSettings)
	if err != nil {
		term.OutputErrorAndExit("Error marshalling settings: %v", err)
		return
	}

	var settings *shared.PlanSettings
	err = json.Unmarshal(jsonBytes, &settings)
	if err != nil {
		term.OutputErrorAndExit("Error unmarshalling settings: %v", err)
		return
	}

	var roleOrSetting, propertyCompact, value string
	var role shared.ModelRole
	var settingCompact string
	var settingDasherized string
	var selectedModel *shared.BaseModelConfig
	var temperature *float64
	var topP *float64

	if len(args) > 0 {
		roleOrSetting = args[0]
		for _, r := range shared.AllModelRoles {
			if strings.EqualFold(string(r), roleOrSetting) {
				role = r
				break
			}
		}
		if role == "" {
			for _, s := range shared.ModelOverridePropsDasherized {
				compact := shared.Compact(s)
				if strings.EqualFold(compact, shared.Compact(roleOrSetting)) {
					settingCompact = compact
					settingDasherized = s
					break
				}
			}
		}
	}

	if role == "" && settingCompact == "" {
		// Prompt user to select between updating a top-level setting or a modelset role
		opts := []string{}
		for _, role := range shared.AllModelRoles {
			label := fmt.Sprintf("🤖 role | %s → %s", role, shared.ModelRoleDescriptions[role])
			opts = append(opts, label)
		}
		for _, setting := range shared.ModelOverridePropsDasherized {
			label := fmt.Sprintf("⚙️  override | %s → %s", shared.Dasherize(setting), shared.SettingDescriptions[setting])
			opts = append(opts, label)
		}

		selection, err := term.SelectFromList("Select a role or override to update:", opts)
		if err != nil {
			if err.Error() == "interrupt" {
				return
			}

			term.OutputErrorAndExit("Error selecting setting or role: %v", err)
			return
		}

		idx := 0
		for i, opt := range opts {
			if opt == selection {
				idx = i
				break
			}
		}

		if idx < len(shared.AllModelRoles) {
			role = shared.AllModelRoles[idx]
		} else {
			settingDasherized = shared.ModelOverridePropsDasherized[idx-len(shared.AllModelRoles)]
			settingCompact = shared.Compact(settingDasherized)

			log.Printf("Selected setting: %s", settingDasherized)
			log.Printf("Selected setting compact: %s", settingCompact)
		}
	}

	if len(args) > 1 {
		if role != "" {
			propertyCompact = shared.Compact(args[1])
		} else {
			value = args[1]
		}
	}

	if len(args) > 2 {
		value = args[2]
	}

	if settingCompact != "" {
		if value == "" {
			var err error
			value, err = term.GetUserStringInput(fmt.Sprintf("Set %s (leave blank for no value)", settingDasherized))
			if err != nil {
				if err.Error() == "interrupt" {
					return
				}

				term.OutputErrorAndExit("Error getting value: %v", err)
				return
			}
		}

		switch settingCompact {
		case "maxconvotokens":
			if value == "" {
				settings.ModelOverrides.MaxConvoTokens = nil
			} else {
				n, err := strconv.Atoi(value)
				if err != nil {
					fmt.Println("Invalid value for max-convo-tokens:", value)
					return
				}
				settings.ModelOverrides.MaxConvoTokens = &n
			}
		case "maxtokens":
			if value == "" {
				settings.ModelOverrides.MaxTokens = nil
			} else {
				n, err := strconv.Atoi(value)
				if err != nil {
					fmt.Println("Invalid value for max-tokens:", value)
					return
				}
				settings.ModelOverrides.MaxTokens = &n
			}
		case "reservedoutputtokens":
			if value == "" {
				settings.ModelOverrides.ReservedOutputTokens = nil
			} else {
				n, err := strconv.Atoi(value)
				if err != nil {
					fmt.Println("Invalid value for reserved-output-tokens:", value)
					return
				}
				settings.ModelOverrides.ReservedOutputTokens = &n
			}
		}
	}

	if role != "" {
		if !(propertyCompact == "temperature" || propertyCompact == "topp") {
			for _, m := range shared.AvailableModels {
				if propertyCompact == m.ModelName {
					selectedModel = &m
					break
				}
			}
		}

		if selectedModel == nil && propertyCompact == "" {
			opts := []string{
				"Select a model",
				"Set temperature",
				"Set top-p",
			}

			selection, err := term.SelectFromList("Select a property to update:", opts)
			if err != nil {
				if err.Error() == "interrupt" {
					return
				}

				term.OutputErrorAndExit("Error selecting property: %v", err)
				return
			}

			if selection == "Select a model" {

				var opts []string
				for _, m := range shared.AvailableModels {
					label := fmt.Sprintf("%s → %s | max %d 🪙", m.Provider, m.ModelName, m.MaxTokens)
					opts = append(opts, label)
				}

				selection, err := term.SelectFromList("Select a model:", opts)

				if err != nil {
					if err.Error() == "interrupt" {
						return
					}

					term.OutputErrorAndExit("Error selecting model: %v", err)
					return
				}

				for i := range opts {
					if opts[i] == selection {
						selectedModel = &shared.AvailableModels[i]
						break
					}
				}

			} else if selection == "Set temperature" {
				propertyCompact = "temperature"
			} else if selection == "Set top-p" {
				propertyCompact = "topp"
			}
		}

		if selectedModel == nil {
			if propertyCompact != "" {
				if value == "" {
					msg := "Set"
					if propertyCompact == "temperature" {
						msg += "temperature (-2.0 to 2.0)"
					} else if propertyCompact == "topp" {
						msg += "top-p (0.0 to 1.0)"
					}
					var err error
					value, err = term.GetUserStringInput(msg)
					if err != nil {
						if err.Error() == "interrupt" {
							return
						}

						term.OutputErrorAndExit("Error getting value: %v", err)
						return
					}
				}

				switch propertyCompact {
				case "temperature":
					f, err := strconv.ParseFloat(value, 32)
					if err != nil || f < -2.0 || f > 2.0 {
						fmt.Println("Invalid value for temperature:", value)
						return
					}
					temperature = &f
				case "topp":
					f, err := strconv.ParseFloat(value, 32)
					if err != nil || f < 0.0 || f > 1.0 {
						fmt.Println("Invalid value for top-p:", value)
						return
					}
					topP = &f
				}
			}
		}

		if settings.ModelSet == nil {
			settings.ModelSet = &shared.DefaultModelSet
		}

		switch role {
		case shared.ModelRolePlanner:
			if selectedModel != nil {
				settings.ModelSet.Planner.BaseModelConfig = *selectedModel
				settings.ModelSet.Planner.PlannerModelConfig = shared.PlannerModelConfigByName[selectedModel.ModelName]
			} else if temperature != nil {
				settings.ModelSet.Planner.Temperature = float32(*temperature)
			} else if topP != nil {
				settings.ModelSet.Planner.TopP = float32(*topP)
			}

		case shared.ModelRolePlanSummary:
			if selectedModel != nil {
				settings.ModelSet.PlanSummary.BaseModelConfig = *selectedModel
			} else if temperature != nil {
				settings.ModelSet.PlanSummary.Temperature = float32(*temperature)
			} else if topP != nil {
				settings.ModelSet.PlanSummary.TopP = float32(*topP)
			}

		case shared.ModelRoleBuilder:
			if selectedModel != nil {
				settings.ModelSet.Builder.BaseModelConfig = *selectedModel
				settings.ModelSet.Builder.TaskModelConfig = shared.TaskModelConfigByName[selectedModel.ModelName]
			} else if temperature != nil {
				settings.ModelSet.Builder.Temperature = float32(*temperature)
			} else if topP != nil {
				settings.ModelSet.Builder.TopP = float32(*topP)
			}

		case shared.ModelRoleName:
			if selectedModel != nil {
				settings.ModelSet.Namer.BaseModelConfig = *selectedModel
				settings.ModelSet.Namer.TaskModelConfig = shared.TaskModelConfigByName[selectedModel.ModelName]
			} else if temperature != nil {
				settings.ModelSet.Namer.Temperature = float32(*temperature)
			} else if topP != nil {
				settings.ModelSet.Namer.TopP = float32(*topP)
			}

		case shared.ModelRoleCommitMsg:
			if selectedModel != nil {
				settings.ModelSet.CommitMsg.BaseModelConfig = *selectedModel
				settings.ModelSet.CommitMsg.TaskModelConfig = shared.TaskModelConfigByName[selectedModel.ModelName]
			} else if temperature != nil {
				settings.ModelSet.CommitMsg.Temperature = float32(*temperature)
			} else if topP != nil {
				settings.ModelSet.CommitMsg.TopP = float32(*topP)
			}

		case shared.ModelRoleExecStatus:
			if selectedModel != nil {
				settings.ModelSet.ExecStatus.BaseModelConfig = *selectedModel
				settings.ModelSet.ExecStatus.TaskModelConfig = shared.TaskModelConfigByName[selectedModel.ModelName]
			} else if temperature != nil {
				settings.ModelSet.ExecStatus.Temperature = float32(*temperature)
			} else if topP != nil {
				settings.ModelSet.ExecStatus.TopP = float32(*topP)
			}
		}
	}

	if reflect.DeepEqual(originalSettings, settings) {
		fmt.Println("🤷‍♂️ No model settings were updated")
		return
	}

	term.StartSpinner("")
	res, apiErr := api.Client.UpdateSettings(
		lib.CurrentPlanId,
		lib.CurrentBranch,
		shared.UpdateSettingsRequest{
			Settings: settings,
		})
	term.StopSpinner()

	if apiErr != nil {
		term.OutputErrorAndExit("Error updating settings: %v", apiErr)
		return
	}

	fmt.Println(res.Msg)
	fmt.Println()
	term.PrintCmds("", "models", "log", "rewind")
}
