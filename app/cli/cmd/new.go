package cmd

import (
	"fmt"

	"plandex/api"
	"plandex/auth"
	"plandex/lib"
	"plandex/term"
	"plandex/types"

	"github.com/fatih/color"
	"github.com/plandex/plandex/shared"
	"github.com/spf13/cobra"
)

var name string
var contextBaseDir string

// newCmd represents the new command
var newCmd = &cobra.Command{
	Use:     "new",
	Aliases: []string{"n"},
	Short:   "Start a new plan",
	// Long:  ``,
	Args: cobra.ExactArgs(0),
	Run:  new,
}

func init() {
	RootCmd.AddCommand(newCmd)
	newCmd.Flags().StringVarP(&name, "name", "n", "", "Name of the new plan")
	newCmd.Flags().StringVar(&contextBaseDir, "context-dir", ".", "Base directory to auto-load context from")
}

func new(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()
	lib.MustResolveOrCreateProject()

	term.StartSpinner("")

	errCh := make(chan error, 2)

	var planId string
	var config *shared.PlanConfig

	go func() {
		res, apiErr := api.Client.CreatePlan(lib.CurrentProjectId, shared.CreatePlanRequest{Name: name})
		if apiErr != nil {
			errCh <- fmt.Errorf("error creating plan: %v", apiErr.Msg)
			return
		}
		planId = res.Id
		errCh <- nil
	}()

	go func() {
		var apiErr *shared.ApiError
		config, apiErr = api.Client.GetDefaultPlanConfig()
		if apiErr != nil {
			errCh <- fmt.Errorf("error getting plan config: %v", apiErr.Msg)
			return
		}
		errCh <- nil
	}()

	for i := 0; i < 2; i++ {
		err := <-errCh
		if err != nil {
			term.OutputErrorAndExit("Error: %v", err)
		}
	}

	err := lib.WriteCurrentPlan(planId)

	if err != nil {
		term.OutputErrorAndExit("Error setting current plan: %v", err)
	}

	err = lib.WriteCurrentBranch("main")
	if err != nil {
		term.OutputErrorAndExit("Error setting current branch: %v", err)
	}

	if name == "" {
		name = "draft"
	}

	term.StopSpinner()

	fmt.Printf("✅ Started new plan %s and set it to current plan\n", color.New(color.Bold, term.ColorHiGreen).Sprint(name))
	fmt.Println("⚙️  Using default config")

	// autoModeLabel := shared.ConfigSettingsByKey["automode"].KeyToLabel(string(config.AutoMode))
	// fmt.Println("⚡️ Auto-mode:", autoModeLabel)

	if config.AutoLoadContext {
		fmt.Println("📥 Automatic context loading is enabled")

		baseDir := contextBaseDir
		if baseDir == "" {
			baseDir = "."
		}

		lib.MustLoadContext([]string{baseDir}, &types.LoadContextParams{
			DefsOnly:          true,
			SkipIgnoreWarning: true,
		})
	} else {
		fmt.Println()
	}

	var cmds []string
	if term.IsRepl {
		cmds = []string{"config", "plans", "cd", "models"}
	} else {
		cmds = []string{"tell", "chat", "config"}
	}

	if !config.AutoLoadContext {
		cmds = append([]string{"load"}, cmds...)
	}

	fmt.Println()
	term.PrintCmds("", cmds...)
}
