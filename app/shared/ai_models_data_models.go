package shared

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

type ModelCompatibility struct {
	HasImageSupport bool `json:"hasImageSupport"`
}

type ModelOutputFormat string

const (
	ModelOutputFormatToolCallJson ModelOutputFormat = "tool-call-json"
	ModelOutputFormatXml          ModelOutputFormat = "xml"
)

const AnthropicMaxReasoningBudget = 32000
const GoogleMaxReasoningBudget = 32000

// to help avoid confusion between the model name and the model id
type ModelName string
type ModelId string

type BaseModelConfig struct {
	Provider                   ModelProvider     `json:"provider"`
	CustomProvider             *string           `json:"customProvider,omitempty"`
	BaseUrl                    string            `json:"baseUrl"`
	ModelName                  ModelName         `json:"modelName"`
	ModelId                    ModelId           `json:"modelId"`
	MaxTokens                  int               `json:"maxTokens"`
	MaxOutputTokens            int               `json:"maxOutputTokens"`
	ReservedOutputTokens       int               `json:"reservedOutputTokens"`
	ApiKeyEnvVar               string            `json:"apiKeyEnvVar"`
	PreferredModelOutputFormat ModelOutputFormat `json:"preferredModelOutputFormat"`
	SystemPromptDisabled       bool              `json:"systemPromptDisabled"`
	RoleParamsDisabled         bool              `json:"roleParamsDisabled"`
	StopDisabled               bool              `json:"stopDisabled"`
	PredictedOutputEnabled     bool              `json:"predictedOutputEnabled"`
	ReasoningEffortEnabled     bool              `json:"reasoningEffortEnabled"`
	ReasoningEffort            ReasoningEffort   `json:"reasoningEffort"`
	IncludeReasoning           bool              `json:"includeReasoning"`

	ReasoningBudgetEnabled bool `json:"reasoningBudgetEnabled"`
	ReasoningBudget        int  `json:"reasoningBudget"`

	SupportsCacheControl bool `json:"supportsCacheControl"`

	// for openai responses API, not fully implemented yet
	UsesOpenAIResponsesAPI bool `json:"usesOpenAIResponsesAPI"`

	// for anthropic, single message system prompt needs to be flipped to 'user'
	SingleMessageNoSystemPrompt bool `json:"singleMessageNoSystemPrompt"`

	// for openai, token estimate padding percentage
	TokenEstimatePaddingPct float64 `json:"tokenEstimatePaddingPct"`

	ModelCompatibility
}

type AvailableModel struct {
	Id string `json:"id"`
	BaseModelConfig
	Description           string    `json:"description"`
	DefaultMaxConvoTokens int       `json:"defaultMaxConvoTokens"`
	CreatedAt             time.Time `json:"createdAt"`
	UpdatedAt             time.Time `json:"updatedAt"`
}

func (m *AvailableModel) ModelString() string {
	s := ""
	if m.Provider != ModelProviderOpenAI {
		s += string(m.Provider) + "/"
	}
	s += string(m.ModelId)
	return s
}

type PlannerModelConfig struct {
	MaxConvoTokens int `json:"maxConvoTokens"`
}

type ReasoningEffort string

const (
	ReasoningEffortLow    ReasoningEffort = "low"
	ReasoningEffortMedium ReasoningEffort = "medium"
	ReasoningEffortHigh   ReasoningEffort = "high"
)

type ModelRoleConfig struct {
	Role                 ModelRole       `json:"role"`
	BaseModelConfig      BaseModelConfig `json:"baseModelConfig"`
	Temperature          float32         `json:"temperature"`
	TopP                 float32         `json:"topP"`
	ReservedOutputTokens int             `json:"reservedOutputTokens"`
	ReasoningEffort      ReasoningEffort `json:"reasoningEffort"`

	LargeContextFallback *ModelRoleConfig `json:"largeContextFallback"`
	LargeOutputFallback  *ModelRoleConfig `json:"largeOutputFallback"`
	ErrorFallback        *ModelRoleConfig `json:"errorFallback"`
	MissingKeyFallback   *ModelRoleConfig `json:"missingKeyFallback"`
	StrongModel          *ModelRoleConfig `json:"strongModel"`
}

func (m ModelRoleConfig) GetReservedOutputTokens() int {
	if m.ReservedOutputTokens > 0 {
		return m.ReservedOutputTokens
	}
	return m.BaseModelConfig.ReservedOutputTokens
}

func (m *ModelRoleConfig) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	switch s := src.(type) {
	case []byte:
		return json.Unmarshal(s, m)
	case string:
		return json.Unmarshal([]byte(s), m)
	default:
		return fmt.Errorf("unsupported data type: %T", src)
	}
}

func (m ModelRoleConfig) Value() (driver.Value, error) {
	return json.Marshal(m)
}

type PlannerRoleConfig struct {
	ModelRoleConfig
	PlannerModelConfig
}

func (p *PlannerRoleConfig) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	switch s := src.(type) {
	case []byte:
		return json.Unmarshal(s, p)
	case string:
		return json.Unmarshal([]byte(s), p)
	default:
		return fmt.Errorf("unsupported data type: %T", src)
	}
}

func (p PlannerRoleConfig) Value() (driver.Value, error) {
	return json.Marshal(p)
}

type ModelPack struct {
	Id               string            `json:"id"`
	Name             string            `json:"name"`
	Description      string            `json:"description"`
	Planner          PlannerRoleConfig `json:"planner"`
	Coder            *ModelRoleConfig  `json:"coder"`
	PlanSummary      ModelRoleConfig   `json:"planSummary"`
	Builder          ModelRoleConfig   `json:"builder"`
	WholeFileBuilder *ModelRoleConfig  `json:"wholeFileBuilder"` // optional, defaults to builder model — access via GetWholeFileBuilder()
	Namer            ModelRoleConfig   `json:"namer"`
	CommitMsg        ModelRoleConfig   `json:"commitMsg"`
	ExecStatus       ModelRoleConfig   `json:"execStatus"`
	Architect        *ModelRoleConfig  `json:"contextLoader"`
}

func (m *ModelPack) GetCoder() ModelRoleConfig {
	if m.Coder == nil {
		return m.Planner.ModelRoleConfig
	}
	return *m.Coder
}

func (m *ModelPack) GetWholeFileBuilder() ModelRoleConfig {
	if m.WholeFileBuilder == nil {
		return m.Builder
	}
	return *m.WholeFileBuilder
}

func (m *ModelPack) GetArchitect() ModelRoleConfig {
	if m.Architect == nil {
		return m.Planner.ModelRoleConfig
	}
	return *m.Architect
}

type ModelOverrides struct {
	MaxConvoTokens       *int `json:"maxConvoTokens"`
	MaxTokens            *int `json:"maxContextTokens"`
	ReservedOutputTokens *int `json:"maxOutputTokens"`
}

type PlanSettings struct {
	ModelOverrides ModelOverrides `json:"modelOverrides"`
	ModelPack      *ModelPack     `json:"modelPack"`
	UpdatedAt      time.Time      `json:"updatedAt"`
}

func (p *PlanSettings) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	switch s := src.(type) {
	case []byte:
		return json.Unmarshal(s, p)
	case string:
		return json.Unmarshal([]byte(s), p)
	default:
		return fmt.Errorf("unsupported data type: %T", src)
	}
}

func (p PlanSettings) Value() (driver.Value, error) {
	return json.Marshal(p)
}
