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

type BaseModelConfig struct {
	Provider                   ModelProvider     `json:"provider"`
	CustomProvider             *string           `json:"customProvider,omitempty"`
	BaseUrl                    string            `json:"baseUrl"`
	ModelName                  string            `json:"modelName"`
	MaxTokens                  int               `json:"maxTokens"`
	ApiKeyEnvVar               string            `json:"apiKeyEnvVar"`
	PreferredModelOutputFormat ModelOutputFormat `json:"preferredModelOutputFormat"`
	// PreferredOpenRouterProviders []OpenRouterProvider `json:"preferredOpenRouterProviders"`
	// OpenRouterAllowFallbacks     bool                 `json:"openRouterAllowFallbacks"`
	SystemPromptDisabled   bool `json:"systemPromptDisabled"`
	RoleParamsDisabled     bool `json:"roleParamsDisabled"`
	PredictedOutputEnabled bool `json:"predictedOutputEnabled"`
	ReasoningEffortEnabled bool `json:"reasoningEffortEnabled"`
	// OpenRouterSelfModerated      bool                 `json:"openRouterSelfModerated"`
	// OpenRouterNitro              bool                 `json:"openRouterNitro"`
	ModelCompatibility
}

type AvailableModel struct {
	Id string `json:"id"`
	BaseModelConfig
	Description                 string    `json:"description"`
	DefaultMaxConvoTokens       int       `json:"defaultMaxConvoTokens"`
	DefaultReservedOutputTokens int       `json:"defaultReservedOutputTokens"`
	CreatedAt                   time.Time `json:"createdAt"`
	UpdatedAt                   time.Time `json:"updatedAt"`
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
	// ErrorFallback        *ModelRoleConfig `json:"errorFallback"`
}

func (m ModelRoleConfig) GetReservedOutputTokens() int {
	if m.ReservedOutputTokens > 0 {
		return m.ReservedOutputTokens
	}
	return GetAvailableModel(m.BaseModelConfig.Provider, m.BaseModelConfig.ModelName).DefaultReservedOutputTokens
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

const maxFallbackDepth = 10 // max fallback depth for large context fallback - should never be reached in real scenarios, but protects against infinite loops in case of circular references etc.

func (m ModelRoleConfig) GetFinalLargeContextFallback() ModelRoleConfig {
	var currentConfig ModelRoleConfig = m
	var n int = 0

	for {
		if currentConfig.LargeContextFallback == nil {
			return currentConfig
		} else {
			currentConfig = *currentConfig.LargeContextFallback
		}
		n++
		if n > maxFallbackDepth {
			break
		}
	}

	return currentConfig
}

func (m ModelRoleConfig) GetFinalLargeOutputFallback() ModelRoleConfig {
	var currentConfig ModelRoleConfig = m
	var n int = 0

	if currentConfig.LargeOutputFallback == nil {
		return currentConfig.GetFinalLargeContextFallback()
	}

	for {
		if currentConfig.LargeOutputFallback == nil {
			return currentConfig
		} else {
			currentConfig = *currentConfig.LargeOutputFallback
		}
		n++
		if n > maxFallbackDepth {
			break
		}
	}

	return currentConfig
}

// note that if the token number exeeds all the fallback models, it will return the last fallback model
func (m ModelRoleConfig) GetRoleForInputTokens(inputTokens int) ModelRoleConfig {
	var currentConfig ModelRoleConfig = m
	var n int = 0
	for {
		if currentConfig.BaseModelConfig.MaxTokens >= inputTokens {
			return currentConfig
		}

		if currentConfig.LargeContextFallback == nil {
			return currentConfig
		} else {
			currentConfig = *currentConfig.LargeContextFallback
		}
		n++
		if n > maxFallbackDepth {
			break
		}
	}
	return currentConfig
}

func (m ModelRoleConfig) GetRoleForOutputTokens(outputTokens int) ModelRoleConfig {
	var currentConfig ModelRoleConfig = m
	var n int = 0
	for {
		if currentConfig.GetReservedOutputTokens() >= outputTokens {
			return currentConfig
		}

		if currentConfig.LargeOutputFallback == nil {
			if currentConfig.LargeContextFallback == nil {
				return currentConfig
			} else {
				currentConfig = *currentConfig.LargeContextFallback
			}
		} else {
			currentConfig = *currentConfig.LargeOutputFallback
		}
		n++
		if n > maxFallbackDepth {
			break
		}
	}
	return currentConfig
}

type PlannerRoleConfig struct {
	ModelRoleConfig
	PlannerModelConfig
	PlannerLargeContextFallback *PlannerRoleConfig `json:"plannerLargeContextFallback"`
	// PlannerErrorFallback        *PlannerRoleConfig `json:"plannerErrorFallback"`
}

func (p PlannerRoleConfig) GetFinalLargeContextFallback() PlannerRoleConfig {
	var currentConfig PlannerRoleConfig = p
	var n int = 0
	for {
		if currentConfig.PlannerLargeContextFallback == nil {
			return currentConfig
		} else {
			currentConfig = *currentConfig.PlannerLargeContextFallback
		}
		n++
		if n > maxFallbackDepth {
			break
		}
	}

	return currentConfig
}

func (p PlannerRoleConfig) GetRoleForTokens(tokens int) PlannerRoleConfig {
	var currentConfig PlannerRoleConfig = p
	var n int = 0
	for {
		if currentConfig.BaseModelConfig.MaxTokens >= tokens {
			return currentConfig
		}

		if currentConfig.PlannerLargeContextFallback == nil {
			return currentConfig
		} else {
			currentConfig = *currentConfig.PlannerLargeContextFallback
		}
		n++
		if n > maxFallbackDepth {
			break
		}
	}
	return currentConfig
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
