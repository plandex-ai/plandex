package cmd

import (
	"fmt"
	"os"
	"plandex-cli/lib"
	"plandex-cli/term"
	shared "plandex-shared"

	"github.com/fatih/color"
)

func warnModelsFileLocalChanges(path, cmd string) (bool, error) {
	cmdPrefix := "\\"
	if !term.IsRepl {
		cmdPrefix = "plandex "
	}

	color.New(color.Bold, term.ColorHiYellow).Println("⚠️  The models file has local changes")

	fmt.Println()
	fmt.Println("Path → " + path)
	fmt.Println()

	fmt.Println("If you continue, local changes will be dropped in favor of the latest server state")

	fmt.Println()
	fmt.Printf("To keep the local version instead, quit and run %s\n", color.New(color.Bold, color.BgCyan, color.FgHiWhite).
		Sprintf(" %s%s --save ", cmdPrefix, cmd))
	fmt.Println()
	return term.ConfirmYesNo("Drop local changes and continue?")

}

type maybePromptAndOpenModelsFileResult struct {
	shouldReturn bool
	jsonData     []byte
}

func maybePromptAndOpenModelsFile(filePath, pathArg, cmd string, defaultConfig *shared.PlanConfig, planConfig *shared.PlanConfig) maybePromptAndOpenModelsFileResult {

	printManual := func() {
		cmdPrefix := "\\"
		if !term.IsRepl {
			cmdPrefix = "plandex "
		}
		fmt.Println("To save changes, run " +
			fmt.Sprintf(" %s ", color.New(color.Bold, color.BgCyan, color.FgHiWhite).
				Sprintf(" %s%s --save%s ", cmdPrefix, cmd, pathArg)))
	}

	selectedEditor := lib.MaybePromptAndOpen(filePath, defaultConfig, planConfig)

	if selectedEditor {
		fmt.Println("📝 Opened in editor")
		fmt.Println()

		confirmed, err := term.ConfirmYesNo("Ready to save?")
		if err != nil {
			term.OutputErrorAndExit("Error confirming: %v", err)
			return maybePromptAndOpenModelsFileResult{shouldReturn: true}
		}

		if !confirmed {
			fmt.Println("🙅‍♂️ Update canceled")
			fmt.Println()

			printManual()
			return maybePromptAndOpenModelsFileResult{shouldReturn: true}
		}

		// get updated file state
		jsonData, err := os.ReadFile(filePath)
		if err != nil {
			term.OutputErrorAndExit("Error reading JSON file: %v", err)
			return maybePromptAndOpenModelsFileResult{shouldReturn: true}
		}

		return maybePromptAndOpenModelsFileResult{shouldReturn: false, jsonData: jsonData}
	} else {
		// No editor available or user chose manual
		fmt.Println("👨‍💻 Edit the file in your JSON editor of choice")
		fmt.Println()

		printManual()
		return maybePromptAndOpenModelsFileResult{shouldReturn: true}
	}
}
