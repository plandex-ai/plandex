package shared

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
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

// to help avoid confusion between model tag, price id, model name, and the model id
type ModelName string
type ModelId string
type ModelTag string
type VariantTag string

type BaseModelShared struct {
	MaxTokens                  int               `json:"maxTokens"`
	MaxOutputTokens            int               `json:"maxOutputTokens"`
	ReservedOutputTokens       int               `json:"reservedOutputTokens"`
	PreferredModelOutputFormat ModelOutputFormat `json:"preferredModelOutputFormat"`
	SystemPromptDisabled       bool              `json:"systemPromptDisabled"`
	RoleParamsDisabled         bool              `json:"roleParamsDisabled"`
	StopDisabled               bool              `json:"stopDisabled"`
	PredictedOutputEnabled     bool              `json:"predictedOutputEnabled"`
	ReasoningEffortEnabled     bool              `json:"reasoningEffortEnabled"`
	ReasoningEffort            ReasoningEffort   `json:"reasoningEffort"`
	IncludeReasoning           bool              `json:"includeReasoning"`
	ReasoningBudget            int               `json:"reasoningBudget"`
	SupportsCacheControl       bool              `json:"supportsCacheControl"`
	// for anthropic, single message system prompt needs to be flipped to 'user'
	SingleMessageNoSystemPrompt bool `json:"singleMessageNoSystemPrompt"`

	// for openai, token estimate padding percentage
	TokenEstimatePaddingPct float64 `json:"tokenEstimatePaddingPct"`

	ModelCompatibility ModelCompatibility `json:"modelCompatibility"`
}

type BaseModelProviderConfig struct {
	ModelProviderConfigSchema
	ModelName ModelName `json:"modelName"`
}

type BaseModelConfig struct {
	ModelId ModelId `json:"modelId"`
	BaseModelShared
	BaseModelProviderConfig
}

type BaseModelUsesProvider struct {
	Provider       ModelProvider `json:"provider"`
	CustomProvider *string       `json:"customProvider,omitempty"`
	ModelName      ModelName     `json:"modelName"`
}

type BaseModelConfigSchema struct {
	ModelTag              ModelTag `json:"modelTag"`
	Description           string   `json:"description"`
	DefaultMaxConvoTokens int      `json:"defaultMaxConvoTokens"`
	BaseModelShared
	Variants  []BaseModelConfigVariant `json:"variants"`
	Providers []BaseModelUsesProvider  `json:"providers"`
}

type BaseModelConfigVariant struct {
	VariantTag  VariantTag `json:"variantTag"`
	Description string     `json:"description"`
	BaseModelShared
	Variants []BaseModelConfigVariant `json:"variants"`
}

func (b *BaseModelConfigSchema) ToAvailableModels() []*AvailableModel {
	avail := []*AvailableModel{}
	for _, provider := range b.Providers {

		providerConfig, ok := BuiltInModelProviderConfigs[provider.Provider]
		if !ok {
			panic(fmt.Sprintf("provider %s not found", provider.Provider))
		}

		avail = append(avail, &AvailableModel{
			Description:           b.Description,
			DefaultMaxConvoTokens: b.DefaultMaxConvoTokens,
			BaseModelConfig: BaseModelConfig{
				ModelId:         ModelId(strings.Join([]string{string(provider.Provider), string(provider.ModelName)}, "/")),
				BaseModelShared: b.BaseModelShared,
				BaseModelProviderConfig: BaseModelProviderConfig{
					ModelProviderConfigSchema: providerConfig,
					ModelName:                 provider.ModelName,
				},
			},
		})
	}
	return avail
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

	LargeContextFallback *ModelRoleConfig `json:"largeContextFallback"`
	LargeOutputFallback  *ModelRoleConfig `json:"largeOutputFallback"`
	ErrorFallback        *ModelRoleConfig `json:"errorFallback"`
	MissingKeyFallback   *ModelRoleConfig `json:"missingKeyFallback"`
	StrongModel          *ModelRoleConfig `json:"strongModel"`
}

type ModelRoleModelConfig struct {
	Provider       ModelProvider `json:"provider"`
	CustomProvider *string       `json:"customProvider,omitempty"`
	ModelTag       ModelTag      `json:"modelTag"`
}

type ModelRoleConfigSchema struct {
	Role     ModelRole `json:"role"`
	ModelTag ModelTag  `json:"modelTag"`

	Temperature          float32 `json:"temperature"`
	TopP                 float32 `json:"topP"`
	ReservedOutputTokens int     `json:"reservedOutputTokens"`
	MaxConvoTokens       int     `json:"maxConvoTokens"`

	LargeContextFallback *ModelRoleConfigSchema `json:"largeContextFallback"`
	LargeOutputFallback  *ModelRoleConfigSchema `json:"largeOutputFallback"`
	ErrorFallback        *ModelRoleConfigSchema `json:"errorFallback"`
	StrongModel          *ModelRoleConfigSchema `json:"strongModel"`
}

func (m *ModelRoleConfigSchema) ToModelRoleConfig() ModelRoleConfig {

	modelSchema

	return m.toModelRoleConfig()
}

func (m *ModelRoleConfigSchema) toModelRoleConfig(providers []BaseModelUsesProvider) ModelRoleConfig {
	if len(providers) == 0 {
		panic("no providers")
	}

	provider := providers[0]
	tail := providers[1:]

	config := GetAvailableModel(m.Provider, m.ModelTag)

	var largeContextFallback *ModelRoleConfig
	if m.LargeContextFallback != nil {
		c := m.LargeContextFallback.ToModelRoleConfig()
		largeContextFallback = &c
	}
	var largeOutputFallback *ModelRoleConfig
	if m.LargeOutputFallback != nil {
		c := m.LargeOutputFallback.ToModelRoleConfig()
		largeOutputFallback = &c
	}
	var errorFallback *ModelRoleConfig
	if m.ErrorFallback != nil {
		c := m.ErrorFallback.ToModelRoleConfig()
		errorFallback = &c
	}
	var strongModel *ModelRoleConfig
	if m.StrongModel != nil {
		c := m.StrongModel.ToModelRoleConfig()
		strongModel = &c
	}

	return ModelRoleConfig{
		Role:                 m.Role,
		BaseModelConfig:      config.BaseModelConfig,
		Temperature:          m.Temperature,
		TopP:                 m.TopP,
		ReservedOutputTokens: m.ReservedOutputTokens,

		LargeContextFallback: largeContextFallback,
		LargeOutputFallback:  largeOutputFallback,
		ErrorFallback:        errorFallback,
		StrongModel:          strongModel,
	}
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

type ModelPackSchema struct {
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
