package shared

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

const defaultAutoDebugTries = 5

const (
	EditorTypeVim  string = "vim"
	EditorTypeNano string = "nano"
)

const defaultEditor = EditorTypeVim

type ReplType string

const (
	ReplTypeTell ReplType = "tell"
	ReplTypeChat ReplType = "chat"
	ReplTypeAuto ReplType = "auto"
)

type AutoModeType string

const (
	AutoModeFull      AutoModeType = "full"
	AutoModeSemi      AutoModeType = "semi"
	AutoModeBasicPlus AutoModeType = "basic-plus"
	AutoModeBasic     AutoModeType = "basic"
	AutoModeNone      AutoModeType = "none"
	AutoModeCustom    AutoModeType = "custom"
)

var AutoModeOptions = [][2]string{
	{string(AutoModeFull), "Full Auto"},

	// "→ automatically selects context, updates changes context, includes only necessary context for each step, continues iterating until plan is complete, builds plan into file edits, applies changes, executes commands, and debugs failed commands up to " + strconv.Itoa(defaultAutoDebugTries) + " attempts."},

	{string(AutoModeSemi), "Semi Auto"},

	// "→  automatically selects context, updates changed context, includes only necessary context for each step, continues iterating until plan is complete, and builds plan into pending file edits. User applies changes manually. Automatically commits after apply in git repos. Plans can include commands to execute, but user must approve execution and debugging of failed commands."},

	{string(AutoModeBasic), "Basic Plus"},

	// " → manual selection of context. Context with changes is updates automatically. Includes only necessary context for each step. Continues iterating until plan is complete, and builds plan into pending file edits. User applies changes manually. Automatically commits after apply in git repos. Plans can include commands to execute, but user must approve execution and debugging of failed commands."},

	{string(AutoModeBasic), "Basic"},

	// " → manual selection of context. Updating changed context must be approved. Each step includes all context (no smart context). Continues iterating until plan is complete, and builds plan into pending file edits. User applies changes manually. No automatic commit after apply in git repos. No execution of commands."},

	{string(AutoModeNone), "None"},

	// "→  manual selection of context. Updating changes context must be approved. Each step includes all context (no smart context). Plans only proceed one iteration at a time with no automatic continuation. Changes are not automatically built into pending file edits. User builds and applies changes manually. No automatic commit after apply in git repos. No execution of commands."},

	{string(AutoModeCustom), "Custom"},

	// " → mix and match config settings individually."},
}

var AutoModeLabels = map[AutoModeType]string{}

// populated in init()
var AutoModeChoices []string

type PlanConfig struct {
	AutoMode AutoModeType `json:"autoMode"`
	// QuietMode bool         `json:"quietMode"`

	Editor       string `json:"editor"`
	AutoContinue bool   `json:"autoContinue"`
	AutoBuild    bool   `json:"autoBuild"`

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

	// ReplMode    bool     `json:"replMode"`
	// DefaultRepl ReplType `json:"defaultRepl"`

	// PlainTextMode     bool `json:"plainTextMode"`
	// PlainTextCommands bool `json:"plainTextCommands"`
	// PlainTextStream   bool `json:"plainTextStream"`
}

// autonomy settings are configured below in init()
var DefaultPlanConfig = PlanConfig{
	Editor: defaultEditor,
}

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

	case AutoModeBasicPlus:
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
	}
}

type ConfigSetting struct {
	Name            string
	Desc            string
	Visible         func(p *PlanConfig) bool
	BoolSetter      func(p *PlanConfig, enabled bool)
	IntSetter       func(p *PlanConfig, value int)
	StringSetter    func(p *PlanConfig, value string)
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
				if option[1] == choice {
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
		Desc: "System editor",
		StringSetter: func(p *PlanConfig, value string) {
			p.Editor = value
		},
		Getter: func(p *PlanConfig) string {
			return p.Editor
		},
		Choices:         &[]string{EditorTypeVim, EditorTypeNano},
		HasCustomChoice: true,
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
}

func init() {
	DefaultPlanConfig.SetAutoMode(AutoModeSemi)

	for _, choice := range AutoModeOptions {
		AutoModeChoices = append(AutoModeChoices, choice[1])
		AutoModeLabels[AutoModeType(choice[0])] = choice[1]
	}

}
