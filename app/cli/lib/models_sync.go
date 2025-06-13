package lib

import (
	"fmt"
	"plandex-cli/term"

	"github.com/fatih/color"
)

func PromptSyncModelsIfNeeded() error {
	var changes []string
	var onApprove []func() error

	customModelsRes, err := CustomModelsCheckLocalChanges(CustomModelsDefaultPath)
	if err != nil {
		return fmt.Errorf("error checking custom models: %v", err)
	}

	if customModelsRes.HasLocalChanges {
		changes = append(
			changes,
			fmt.Sprintf("%s → %s", color.New(term.ColorHiCyan, color.Bold).Sprint("Custom models"), CustomModelsDefaultPath))

		onApprove = append(onApprove, SyncCustomModels)
	}

	defaultModelSettingsRes, err := ModelSettingsCheckLocalChanges(DefaultModelSettingsPath)
	if err != nil {
		return fmt.Errorf("error checking default model settings: %v", err)
	}

	if defaultModelSettingsRes.HasLocalChanges {
		changes = append(
			changes,
			fmt.Sprintf("%s → %s", color.New(term.ColorHiCyan, color.Bold).Sprint("Default model settings"), DefaultModelSettingsPath))

		onApprove = append(onApprove, SyncDefaultModelSettings)
	}

	planModelSettingsRes, err := ModelSettingsCheckLocalChanges(GetPlanModelSettingsPath(CurrentPlanId))
	if err != nil {
		return fmt.Errorf("error checking plan model settings: %v", err)
	}

	if planModelSettingsRes.HasLocalChanges {
		changes = append(
			changes,
			fmt.Sprintf("%s → %s", color.New(term.ColorHiCyan, color.Bold).Sprint("Plan model settings"), GetPlanModelSettingsPath(CurrentPlanId)))

		onApprove = append(onApprove, SyncPlanModelSettings)
	}

	if len(changes) == 0 {
		return nil
	}

	term.StopSpinner()
	color.New(color.Bold, term.ColorHiYellow).Println("⚠️  Model settings have local changes")

	fmt.Println()
	for _, change := range changes {
		fmt.Println(change)
	}
	fmt.Println()

	shouldSave, err := term.ConfirmYesNo("Save changes now?")
	if err != nil {
		return fmt.Errorf("error confirming: %v", err)
	}

	if !shouldSave {
		return nil
	}

	for _, fn := range onApprove {
		err := fn()
		if err != nil {
			return fmt.Errorf("error syncing models: %v", err)
		}
	}

	return nil
}
