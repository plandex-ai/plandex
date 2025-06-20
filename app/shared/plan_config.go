package shared

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
)

const defaultAutoDebugTries = 5

const (
	EditorTypeVim  string = "vim"
	EditorTypeNano string = "nano"
)

const defaultEditor = EditorTypeVim

type AutoModeType string

const (
	AutoModeFull   AutoModeType = "full"
	AutoModeSemi   AutoModeType = "semi"
	AutoModePlus   AutoModeType = "plus"
	AutoModeBasic  AutoModeType = "basic"
	AutoModeNone   AutoModeType = "none"
	AutoModeCustom AutoModeType = "custom"
)

var AutoModeDescriptions = map[AutoModeType]string{
	AutoModeFull:   "Fully automated: context, apply, execution and debugging",
	AutoModeSemi:   "Auto context, manual apply and execution",
	AutoModePlus:   "Manual context with auto updates and smart loading, manual apply and execution",
	AutoModeBasic:  "Manual context, manual apply and execution",
	AutoModeNone:   "Fully manual and step-by-step, one response at a time, manual builds",
	AutoModeCustom: "Choose settings individually with set-config command",
}

var AutoModeOptions = [][3]string{
	{string(AutoModeFull), "Full Auto", AutoModeDescriptions[AutoModeFull]},
	{string(AutoModeSemi), "Semi Auto", AutoModeDescriptions[AutoModeSemi]},
	{string(AutoModePlus), "Basic Plus", AutoModeDescriptions[AutoModePlus]},
	{string(AutoModeBasic), "Basic", AutoModeDescriptions[AutoModeBasic]},
	{string(AutoModeNone), "None", AutoModeDescriptions[AutoModeNone]},
	{string(AutoModeCustom), "Custom", AutoModeDescriptions[AutoModeCustom]},
}

var AutoModeLabels = map[AutoModeType]string{}

// populated in init()
var AutoModeChoices []string

type PlanConfig struct {
	AutoMode AutoModeType `json:"autoMode"`
	// QuietMode bool         `json:"quietMode"`

	Editor             string   `json:"editor"`
	EditorCommand      string   `json:"editorCommand"`
	EditorArgs         []string `json:"editorArgs"`
	EditorOpenManually bool     `json:"editorOpenManually"`

	AutoContinue bool `json:"autoContinue"`
	AutoBuild    bool `json:"autoBuild"`

	AutoUpdateContext bool `json:"autoUpdateContext"`
	AutoLoadContext   bool `json:"autoContext"`
	SmartContext      bool `json:"smartContext"`

	// AutoApproveContext bool `json:"autoApproveContext"`
	// QuietContext       bool `json:"quietContext"`

	// AutoApprovePlan bool `json:"autoApprovePlan"`

	// QuietCoding    bool `json:"quietCoding"`
	// ParallelCoding bool `json:"parallelCoding"`

	AutoApply  bool `json:"autoApply"`
	AutoCommit bool `json:"autoCommit"`
	SkipCommit bool `json:"skipCommit"`

	CanExec        bool `json:"canExec"`
	AutoExec       bool `json:"autoExec"`
	AutoDebug      bool `json:"autoDebug"`
	AutoDebugTries int  `json:"autoDebugTries"`

	AutoRevertOnRewind bool `json:"autoRevertOnRewind"`

	// ReplMode    bool     `json:"replMode"`
	// DefaultRepl ReplType `json:"defaultRepl"`

	// PlainTextMode     bool `json:"plainTextMode"`
	// PlainTextCommands bool `json:"plainTextCommands"`
	// PlainTextStream   bool `json:"plainTextStream"`
}

var DefaultPlanConfig = PlanConfig{}

func (p *PlanConfig) Scan(src interface{}) error {
	if src == nil {
		*p = DefaultPlanConfig
		return nil
	}
	switch s := src.(type) {
	case []byte:
		if len(s) == 0 {
			*p = DefaultPlanConfig
			return nil
		}
		return json.Unmarshal(s, p)
	case string:
		if s == "" {
			*p = DefaultPlanConfig
			return nil
		}
		return json.Unmarshal([]byte(s), p)
	default:
		return fmt.Errorf("unsupported data type: %T", src)
	}
}

func (p PlanConfig) Value() (driver.Value, error) {
	return json.Marshal(p)
}

func (p *PlanConfig) SetAutoMode(mode AutoModeType) {
	p.AutoMode = mode

	switch p.AutoMode {
	case AutoModeFull:
		p.AutoContinue = true
		p.AutoBuild = true
		p.AutoUpdateContext = true
		p.AutoLoadContext = true
		p.SmartContext = true
		p.AutoApply = true
		p.AutoCommit = true
		p.CanExec = true
		p.AutoExec = true
		p.AutoDebug = true
		p.AutoDebugTries = defaultAutoDebugTries
		p.AutoRevertOnRewind = true

	case AutoModeSemi:
		p.AutoContinue = true
		p.AutoBuild = true
		p.AutoUpdateContext = true
		p.AutoLoadContext = true
		p.SmartContext = true
		p.AutoApply = false
		p.AutoCommit = true
		p.CanExec = true
		p.AutoExec = false
		p.AutoDebug = false
		p.AutoRevertOnRewind = true

	case AutoModePlus:
		p.AutoContinue = true
		p.AutoBuild = true
		p.AutoUpdateContext = true
		p.AutoLoadContext = false
		p.SmartContext = true
		p.AutoApply = false
		p.AutoCommit = true
		p.CanExec = true
		p.AutoExec = false
		p.AutoDebug = false
		p.AutoRevertOnRewind = true

	case AutoModeBasic:
		p.AutoContinue = true
		p.AutoBuild = true
		p.AutoUpdateContext = false
		p.AutoLoadContext = false
		p.SmartContext = false
		p.AutoApply = false
		p.AutoCommit = false
		p.CanExec = false
		p.AutoExec = false
		p.AutoDebug = false
		p.AutoRevertOnRewind = true

	case AutoModeNone:
		p.AutoContinue = false
		p.AutoBuild = false
		p.AutoUpdateContext = false
		p.AutoLoadContext = false
		p.SmartContext = false
		p.AutoApply = false
		p.AutoCommit = false
		p.CanExec = false
		p.AutoExec = false
		p.AutoDebug = false
		p.AutoRevertOnRewind = true
	}
}

type ConfigSetting struct {
	Name            string
	Desc            string
	Visible         func(p *PlanConfig) bool
	BoolSetter      func(p *PlanConfig, enabled bool)
	IntSetter       func(p *PlanConfig, value int)
	StringSetter    func(p *PlanConfig, value string)
	EditorSetter    func(p *PlanConfig, label, command string, args []string)
	Getter          func(p *PlanConfig) string
	Choices         *[]string
	HasCustomChoice bool
	ChoiceToKey     func(choice string) string
	SortKey         string
	KeyToLabel      func(key string) string
}

var ConfigSettingsByKey = map[string]ConfigSetting{

	"automode": {
		Name: "auto-mode",
		Desc: "Use preset config based on desired level of autonomy",
		StringSetter: func(p *PlanConfig, value string) {
			p.SetAutoMode(AutoModeType(value))
		},
		Getter: func(p *PlanConfig) string {
			return string(p.AutoMode)
		},
		Choices: &AutoModeChoices,
		ChoiceToKey: func(choice string) string {
			for _, option := range AutoModeOptions {
				if strings.HasPrefix(choice, option[1]) {
					return option[0]
				}
			}
			return ""
		},
		KeyToLabel: func(key string) string {
			for _, option := range AutoModeOptions {
				if option[0] == key {
					return option[1]
				}
			}
			return ""
		},
		SortKey: "0",
	},

	"editor": {
		Name: "editor",
		Desc: "Preferred editor",
		EditorSetter: func(p *PlanConfig, label, command string, args []string) {
			p.Editor = label
			p.EditorCommand = command
			p.EditorArgs = args
		},
		Getter: func(p *PlanConfig) string {
			return p.Editor
		},
	},

	"autocontinue": {
		Name: "auto-continue",
		Desc: "Continue iterating until plan is complete",
		BoolSetter: func(p *PlanConfig, enabled bool) {
			if enabled != p.AutoContinue {
				p.AutoMode = AutoModeCustom
			}

			p.AutoContinue = enabled
		},
		Getter: func(p *PlanConfig) string {
			return fmt.Sprintf("%t", p.AutoContinue)
		},
	},

	"autobuild": {
		Name: "auto-build",
		Desc: "Automatically generate pending file edits",
		BoolSetter: func(p *PlanConfig, enabled bool) {
			if enabled != p.AutoBuild {
				p.AutoMode = AutoModeCustom
			}

			p.AutoBuild = enabled

			if !enabled {
				p.AutoApply = false
			}
		},
		Getter: func(p *PlanConfig) string {
			return fmt.Sprintf("%t", p.AutoBuild)
		},
	},
	"autoupdatecontext": {
		Name: "auto-update-context",
		Desc: "Automatically update context after changes",
		BoolSetter: func(p *PlanConfig, enabled bool) {
			if enabled != p.AutoUpdateContext {
				p.AutoMode = AutoModeCustom
			}
			p.AutoUpdateContext = enabled
		},
		Getter: func(p *PlanConfig) string {
			return fmt.Sprintf("%t", p.AutoUpdateContext)
		},
	},
	"autoloadcontext": {
		Name: "auto-load-context",
		Desc: "Find and load context automatically",
		BoolSetter: func(p *PlanConfig, enabled bool) {
			if enabled != p.AutoLoadContext {
				p.AutoMode = AutoModeCustom
			}
			p.AutoLoadContext = enabled
		},
		Getter: func(p *PlanConfig) string {
			return fmt.Sprintf("%t", p.AutoLoadContext)
		},
	},
	"smartcontext": {
		Name: "smart-context",
		Desc: "Load only necessary context for each task in the plan",
		BoolSetter: func(p *PlanConfig, enabled bool) {
			if enabled != p.SmartContext {
				p.AutoMode = AutoModeCustom
			}
			p.SmartContext = enabled
		},
		Getter: func(p *PlanConfig) string {
			return fmt.Sprintf("%t", p.SmartContext)
		},
	},
	"autocommit": {
		Name: "auto-commit",
		Desc: "Automatically commit changes to git after apply",
		BoolSetter: func(p *PlanConfig, enabled bool) {
			if enabled != p.AutoCommit {
				p.AutoMode = AutoModeCustom
			}
			p.AutoCommit = enabled
		},
		Getter: func(p *PlanConfig) string {
			return fmt.Sprintf("%t", p.AutoCommit)
		},
	},
	"skipcommit": {
		Name: "skip-commit",
		Desc: "Skip committing changes to git after apply",
		BoolSetter: func(p *PlanConfig, enabled bool) {
			p.SkipCommit = enabled
		},
		Getter: func(p *PlanConfig) string {
			return fmt.Sprintf("%t", p.SkipCommit)
		},
	},
	"autoapply": {
		Name: "auto-apply",
		Desc: "Automatically apply changes after plan finishes",
		BoolSetter: func(p *PlanConfig, enabled bool) {
			if enabled != p.AutoApply {
				p.AutoMode = AutoModeCustom
			}
			p.AutoApply = enabled

			if enabled {
				p.AutoBuild = true
			}
		},
		Getter: func(p *PlanConfig) string {
			return fmt.Sprintf("%t", p.AutoApply)
		},
	},
	"canexec": {
		Name: "can-exec",
		Desc: "Allow execution of commands",
		BoolSetter: func(p *PlanConfig, enabled bool) {
			if enabled != p.CanExec {
				p.AutoMode = AutoModeCustom
			}
			p.CanExec = enabled

			if !enabled {
				p.AutoExec = false
				p.AutoDebug = false
			}
		},
		Getter: func(p *PlanConfig) string {
			return fmt.Sprintf("%t", p.CanExec)
		},
	},
	"autoexec": {
		Name: "auto-exec",
		Desc: "Automatically execute commands after plan is applied",
		Visible: func(p *PlanConfig) bool {
			return p.CanExec
		},
		BoolSetter: func(p *PlanConfig, enabled bool) {
			if enabled != p.AutoExec {
				p.AutoMode = AutoModeCustom
			}
			p.AutoExec = enabled

			if enabled {
				p.CanExec = true
			} else {
				p.AutoDebug = false
			}
		},
		Getter: func(p *PlanConfig) string {
			return fmt.Sprintf("%t", p.AutoExec)
		},
	},
	"autodebug": {
		Name: "auto-debug",
		Desc: "Automatically debug failed commands",
		Visible: func(p *PlanConfig) bool {
			return p.AutoExec
		},
		BoolSetter: func(p *PlanConfig, enabled bool) {
			if enabled != p.AutoDebug {
				p.AutoMode = AutoModeCustom
			}
			p.AutoDebug = enabled

			if enabled {
				p.CanExec = true
				p.AutoExec = true

				if p.AutoDebugTries == 0 {
					p.AutoDebugTries = defaultAutoDebugTries
				}
			}
		},
		Getter: func(p *PlanConfig) string {
			return fmt.Sprintf("%t", p.AutoDebug)
		},
	},
	"autodebugtries": {
		Name: "auto-debug-tries",
		Desc: "Number of auto-debug attempts",
		Visible: func(p *PlanConfig) bool {
			return p.AutoDebug
		},
		IntSetter: func(p *PlanConfig, value int) {
			if value != p.AutoDebugTries {
				p.AutoMode = AutoModeCustom
			}
			p.AutoDebugTries = value

			if p.AutoDebugTries == 0 {
				p.AutoDebug = false
			} else {
				p.CanExec = true
				p.AutoExec = true
				p.AutoDebug = true
			}
		},
		Getter: func(p *PlanConfig) string {
			return fmt.Sprintf("%d", p.AutoDebugTries)
		},
	},
	"autorevert": {
		Name: "auto-revert",
		Desc: "Automatically update project files when rewinding plan",
		BoolSetter: func(p *PlanConfig, enabled bool) {
			if enabled != p.AutoRevertOnRewind {
				p.AutoMode = AutoModeCustom
			}
			p.AutoRevertOnRewind = enabled
		},
		Getter: func(p *PlanConfig) string {
			return fmt.Sprintf("%t", p.AutoRevertOnRewind)
		},
	},
}

func init() {
	DefaultPlanConfig.SetAutoMode(AutoModeSemi)

	for _, choice := range AutoModeOptions {
		AutoModeChoices = append(AutoModeChoices, fmt.Sprintf("%s → %s", choice[1], choice[2]))
		AutoModeLabels[AutoModeType(choice[0])] = choice[1]
	}
}
