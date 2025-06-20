package cmd

import (
	"fmt"
	"os"
	"plandex-cli/api"
	"plandex-cli/lib"
	"plandex-cli/term"
	"strconv"

	shared "plandex-shared"

	"github.com/spf13/cobra"
)

const (
	EditorTypeVim  string = "vim"
	EditorTypeNano string = "nano"
)

var defaultEditor = EditorTypeVim

const defaultAutoDebugTries = 5

var autoConfirm bool

var tellPromptFile string
var tellBg bool
var tellStop bool
var tellNoBuild bool
var tellAutoApply bool
var tellAutoContext bool
var tellSmartContext bool
var noExec bool
var autoDebug int

var editor = EditorTypeVim // default to vim
var editorSetByFlag bool

func init() {
	envEditor := os.Getenv("EDITOR")
	if envEditor == "" {
		envEditor = os.Getenv("VISUAL")
	}

	if envEditor != "" {
		defaultEditor = envEditor
	}
}

type initExecFlagsParams struct {
	omitFile         bool
	omitNoBuild      bool
	omitEditor       bool
	omitStop         bool
	omitBg           bool
	omitApply        bool
	omitExec         bool
	omitAutoContext  bool
	omitSmartContext bool
}

func initExecFlags(cmd *cobra.Command, params initExecFlagsParams) {
	if !params.omitFile {
		cmd.Flags().StringVarP(&tellPromptFile, "file", "f", "", "File containing prompt")
	}

	if !params.omitBg {
		cmd.Flags().BoolVar(&tellBg, "bg", false, "Execute autonomously in the background")
	}

	if !params.omitStop {
		cmd.Flags().BoolVarP(&tellStop, "stop", "s", false, "Stop after a single reply")
	}

	if !params.omitNoBuild {
		cmd.Flags().BoolVarP(&tellNoBuild, "no-build", "n", false, "Don't build files")
	}

	cmd.Flags().BoolVar(&autoConfirm, "auto-update-context", false, shared.ConfigSettingsByKey["auto-update-context"].Desc)

	if !params.omitAutoContext {
		cmd.Flags().BoolVar(&tellAutoContext, "auto-load-context", false, shared.ConfigSettingsByKey["auto-load-context"].Desc)
	}

	if !params.omitSmartContext {
		cmd.Flags().BoolVar(&tellSmartContext, "smart-context", false, shared.ConfigSettingsByKey["smart-context"].Desc)
	}

	if !params.omitApply {
		cmd.Flags().BoolVar(&tellAutoApply, "apply", false, "Automatically apply changes")
		initApplyFlags(cmd, true)
	}

	if !params.omitExec {
		initExecScriptFlags(cmd)
	}

	if !params.omitEditor {
		cmd.Flags().Var(newEditorValue(&editor), "editor", "Write prompt in system editor")
		cmd.Flag("editor").NoOptDefVal = defaultEditor
	}
}

func initApplyFlags(cmd *cobra.Command, applyFlag bool) {
	commitDesc := "Commit changes to git"
	if applyFlag {
		commitDesc += " when --apply is passed"
	}

	skipCommitDesc := "Skip committing changes to git"
	if applyFlag {
		skipCommitDesc += " when --apply is passed"
	}
	cmd.Flags().BoolVarP(&autoCommit, "commit", "c", false, commitDesc)
	cmd.Flags().BoolVar(&skipCommit, "skip-commit", false, skipCommitDesc)
}

func initExecScriptFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&noExec, "no-exec", false, "Disable command execution")
	cmd.Flags().BoolVar(&autoExec, "auto-exec", false, "Automatically execute commands without confirmation")
	cmd.Flags().Var(newAutoDebugValue(&autoDebug), "debug", "Automatically execute and debug failing commands (optionally specify number of tries—default is 5)")
	cmd.Flag("debug").NoOptDefVal = strconv.Itoa(defaultAutoDebugTries)
}

func validatePlanExecFlags() {
	if tellAutoApply && tellNoBuild {
		term.OutputErrorAndExit("--apply can't be used with --no-build/-n")
	}
	if tellAutoApply && tellBg {
		term.OutputErrorAndExit("--apply can't be used with --bg")
	}
	if autoExec && !tellAutoApply {
		term.OutputErrorAndExit("--auto-exec can only be used with --apply")
	}
	if autoDebug > 0 && !tellAutoApply {
		term.OutputErrorAndExit("--debug can only be used with --apply")
	}
	if autoDebug > 0 && noExec {
		term.OutputErrorAndExit("--debug can't be used with --no-exec")
	}

	if tellAutoContext && tellBg {
		term.OutputErrorAndExit("--auto-context/-c can't be used with --bg")
	}
}

func mustSetPlanExecFlags(cmd *cobra.Command) {
	mustSetPlanExecFlagsWithConfig(cmd, nil)
}

func mustSetPlanExecFlagsWithConfig(cmd *cobra.Command, config *shared.PlanConfig) {
	if lib.CurrentPlanId == "" {
		term.OutputNoCurrentPlanErrorAndExit()
	}

	if config == nil {
		var apiErr *shared.ApiError
		config, apiErr = api.Client.GetPlanConfig(lib.CurrentPlanId)
		if apiErr != nil {
			term.OutputErrorAndExit("Error getting plan config: %v", apiErr)
		}
	}

	// Set flag vars from config when flags aren't explicitly set
	if !cmd.Flags().Changed("stop") {
		tellStop = !config.AutoContinue
	}

	if !cmd.Flags().Changed("no-build") {
		tellNoBuild = !config.AutoBuild
	}

	if !cmd.Flags().Changed("auto-update-context") {
		autoConfirm = config.AutoUpdateContext
	}
	if !cmd.Flags().Changed("apply") {
		tellAutoApply = config.AutoApply
	}
	if !cmd.Flags().Changed("skip-commit") {
		skipCommit = config.SkipCommit
	}
	if !cmd.Flags().Changed("commit") {
		autoCommit = config.AutoCommit
	}
	if !cmd.Flags().Changed("auto-load-context") {
		tellAutoContext = config.AutoLoadContext
	}
	if !cmd.Flags().Changed("smart-context") {
		tellSmartContext = config.SmartContext
	}
	if !cmd.Flags().Changed("no-exec") {
		noExec = !config.CanExec
	}
	if !cmd.Flags().Changed("auto-exec") {
		autoExec = config.AutoExec
	}
	if !cmd.Flags().Changed("debug") {
		autoDebug = config.AutoDebugTries
		// Only set autoDebug if AutoDebug is enabled in config
		if !config.AutoDebug {
			autoDebug = 0
		}
	}

	// tell command editor is no longer tied to config *unless* it's set to vim or nano
	// otherwise, the flag or EDITOR env var are used
	// config.Editor is now used for mainly for JSON editing (and perhaps other purposes)
	// this is because it's pretty rare to use the editor for writing prompts now rather than the REPL
	if !editorSetByFlag && (config.Editor == shared.EditorTypeVim || config.Editor == shared.EditorTypeNano) {
		editor = config.Editor
	}

	validatePlanExecFlags()
}

// AutoDebugValue implements the flag.Value interface
type autoDebugValue struct {
	value *int
}

func newAutoDebugValue(p *int) *autoDebugValue {
	*p = 0 // Default to 0 (disabled)
	return &autoDebugValue{p}
}

func (f *autoDebugValue) Set(s string) error {
	if s == "" {
		*f.value = defaultAutoDebugTries
		return nil
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return fmt.Errorf("invalid value for --debug: %v", err)
	}
	if v <= 0 {
		return fmt.Errorf("--debug value must be greater than 0")
	}
	*f.value = v
	return nil
}

func (f *autoDebugValue) String() string {
	if f.value == nil {
		return "0"
	}
	return strconv.Itoa(*f.value)
}

func (f *autoDebugValue) Type() string {
	return "int"
}

// EditorValue implements the flag.Value interface
type editorValue struct {
	value *string
}

func newEditorValue(p *string) *editorValue {
	*p = defaultEditor
	return &editorValue{p}
}

func (f *editorValue) Set(s string) error {
	if s == "" {
		*f.value = defaultEditor
		return nil
	}
	*f.value = s
	editorSetByFlag = true
	return nil
}

func (f *editorValue) String() string {
	if f.value == nil {
		return ""
	}
	return *f.value
}

func (f *editorValue) Type() string {
	return "string"
}
