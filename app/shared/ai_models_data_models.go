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
	DefaultMaxConvoTokens  int               `json:"defaultMaxConvoTokens"`
	MaxTokens              int               `json:"maxTokens"`
	MaxOutputTokens        int               `json:"maxOutputTokens"`
	ReservedOutputTokens   int               `json:"reservedOutputTokens"`
	PreferredOutputFormat  ModelOutputFormat `json:"preferredOutputFormat"`
	SystemPromptDisabled   bool              `json:"systemPromptDisabled"`
	RoleParamsDisabled     bool              `json:"roleParamsDisabled"`
	StopDisabled           bool              `json:"stopDisabled"`
	PredictedOutputEnabled bool              `json:"predictedOutputEnabled"`
	ReasoningEffortEnabled bool              `json:"reasoningEffortEnabled"`
	ReasoningEffort        ReasoningEffort   `json:"reasoningEffort"`
	IncludeReasoning       bool              `json:"includeReasoning"`
	ReasoningBudget        int               `json:"reasoningBudget"`
	SupportsCacheControl   bool              `json:"supportsCacheControl"`
	// for anthropic, single message system prompt needs to be flipped to 'user'
	SingleMessageNoSystemPrompt bool `json:"singleMessageNoSystemPrompt"`

	// for anthropic, token estimate padding percentage
	TokenEstimatePaddingPct float64 `json:"tokenEstimatePaddingPct"`

	ModelCompatibility
}

type BaseModelProviderConfig struct {
	ModelProviderConfigSchema
	ModelName ModelName `json:"modelName"`
}

type BaseModelConfig struct {
	ModelId   ModelId        `json:"modelId"`
	ModelTag  ModelTag       `json:"modelTag"`
	Publisher ModelPublisher `json:"publisher"`
	BaseModelShared
	BaseModelProviderConfig
}

type BaseModelUsesProvider struct {
	Provider       ModelProvider `json:"provider"`
	CustomProvider *string       `json:"customProvider,omitempty"`
	ModelName      ModelName     `json:"modelName"`
}

func (b BaseModelUsesProvider) ToComposite() string {
	if b.CustomProvider != nil {
		return fmt.Sprintf("%s|%s", b.Provider, *b.CustomProvider)
	}
	return string(b.Provider)
}

type BaseModelConfigSchema struct {
	ModelTag    ModelTag       `json:"modelTag"`
	ModelId     ModelId        `json:"modelId"`
	Publisher   ModelPublisher `json:"publisher"`
	Description string         `json:"description"`

	BaseModelShared

	RequiresVariantOverrides []string `json:"requiresVariantOverrides"`

	Variants  []BaseModelConfigVariant `json:"variants"`
	Providers []BaseModelUsesProvider  `json:"providers"`
}

type BaseModelConfigVariant struct {
	IsBaseVariant            bool                     `json:"isBaseVariant"`
	VariantTag               VariantTag               `json:"variantTag"`
	Description              string                   `json:"description"`
	Overrides                BaseModelShared          `json:"overrides"`
	Variants                 []BaseModelConfigVariant `json:"variants"`
	RequiresVariantOverrides []string                 `json:"requiresVariantOverrides"`
}

func (b *BaseModelConfigSchema) ToAvailableModels() []*AvailableModel {
	avail := []*AvailableModel{}
	for _, provider := range b.Providers {

		providerConfig, ok := BuiltInModelProviderConfigs[provider.Provider]
		if !ok {
			panic(fmt.Sprintf("provider %s not found", provider.Provider))
		}

		addBase := func() {
			avail = append(avail, &AvailableModel{
				Description:           b.Description,
				DefaultMaxConvoTokens: b.DefaultMaxConvoTokens,
				BaseModelConfig: BaseModelConfig{
					ModelTag:        b.ModelTag,
					ModelId:         ModelId(string(b.ModelTag)),
					BaseModelShared: b.BaseModelShared,
					BaseModelProviderConfig: BaseModelProviderConfig{
						ModelProviderConfigSchema: providerConfig,
						ModelName:                 provider.ModelName,
					},
				},
			})
		}

		type variantParams struct {
			BaseVariant              *BaseModelConfigVariant
			BaseId                   ModelId
			BaseDescription          string
			Overrides                BaseModelShared
			RequiresVariantOverrides []string
		}

		addBaseVariant := func(params variantParams) {
			if params.BaseVariant == nil {
				addBase()
				return
			}
			baseDescription := params.BaseVariant.Description
			baseId := params.BaseId
			mergedOverrides := params.Overrides

			avail = append(avail, &AvailableModel{
				Description:           baseDescription,
				DefaultMaxConvoTokens: b.DefaultMaxConvoTokens,
				BaseModelConfig: BaseModelConfig{
					ModelTag:        b.ModelTag,
					ModelId:         baseId,
					Publisher:       b.Publisher,
					BaseModelShared: mergedOverrides,
					BaseModelProviderConfig: BaseModelProviderConfig{
						ModelProviderConfigSchema: providerConfig,
						ModelName:                 provider.ModelName,
					},
				},
			})
		}

		if len(b.Variants) == 0 {
			addBase()
		} else {

			var addVariants func(variants []BaseModelConfigVariant, baseParams variantParams)
			addVariants = func(variants []BaseModelConfigVariant, baseParams variantParams) {
				for _, variant := range variants {
					if variant.IsBaseVariant {
						addBaseVariant(baseParams)
						continue
					}

					if len(baseParams.RequiresVariantOverrides) > 0 {
						ok, missing := FieldsDefined(variant.Overrides, baseParams.RequiresVariantOverrides)
						if !ok {
							panic(fmt.Sprintf("variant %s is missing required field %s", variant.VariantTag, missing))
						}
					}

					var baseId ModelId
					var baseDescription string
					if baseParams.BaseId != "" {
						baseId = baseParams.BaseId
						baseDescription = baseParams.BaseDescription
					} else {
						baseId = ModelId(string(b.ModelTag))
						baseDescription = b.Description
					}

					modelId := ModelId(strings.Join([]string{string(baseId), string(variant.VariantTag)}, "-"))

					description := strings.Join([]string{baseDescription, variant.Description}, " ")

					merged := Merge(b.BaseModelShared, variant.Overrides)

					if len(variant.Variants) > 0 {
						addVariants(variant.Variants, variantParams{
							BaseVariant:              &variant,
							BaseId:                   modelId,
							BaseDescription:          description,
							Overrides:                merged,
							RequiresVariantOverrides: variant.RequiresVariantOverrides,
						})
					}
					avail = append(avail, &AvailableModel{
						Description:           description,
						DefaultMaxConvoTokens: b.DefaultMaxConvoTokens,
						BaseModelConfig: BaseModelConfig{
							ModelTag:        b.ModelTag,
							ModelId:         modelId,
							BaseModelShared: merged,
							BaseModelProviderConfig: BaseModelProviderConfig{
								ModelProviderConfigSchema: providerConfig,
								ModelName:                 provider.ModelName,
							},
						},
					})
				}
			}

			addVariants(b.Variants, variantParams{
				BaseId:                   ModelId(string(b.ModelTag)),
				BaseDescription:          b.Description,
				Overrides:                b.BaseModelShared,
				RequiresVariantOverrides: b.RequiresVariantOverrides,
			})
		}
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
	Role ModelRole `json:"role"`

	ModelId ModelId `json:"modelId"` // new in 2.2.0 refactor — uses provider lookup instead of BaseModelConfig and MissingKeyFallback

	BaseModelConfig      *BaseModelConfig `json:"baseModelConfig,omitempty"`
	Temperature          float32          `json:"temperature"`
	TopP                 float32          `json:"topP"`
	ReservedOutputTokens int              `json:"reservedOutputTokens"`

	LargeContextFallback *ModelRoleConfig `json:"largeContextFallback"`
	LargeOutputFallback  *ModelRoleConfig `json:"largeOutputFallback"`
	ErrorFallback        *ModelRoleConfig `json:"errorFallback"`
	// MissingKeyFallback   *ModelRoleConfig `json:"missingKeyFallback"` // removed in 2.2.0 refactor —
	StrongModel *ModelRoleConfig `json:"strongModel"`
}

type ModelRoleModelConfig struct {
	Provider       ModelProvider `json:"provider"`
	CustomProvider *string       `json:"customProvider,omitempty"`
	ModelTag       ModelTag      `json:"modelTag"`
}

type ModelRoleConfigSchema struct {
	Role    ModelRole `json:"role"`
	ModelId ModelId   `json:"modelId"`

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

	return m.toModelRoleConfig()
}

func (m *ModelRoleConfigSchema) toModelRoleConfig() ModelRoleConfig {
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
		Role: m.Role,

		ModelId: m.ModelId,

		Temperature:          m.Temperature,
		TopP:                 m.TopP,
		ReservedOutputTokens: m.ReservedOutputTokens,

		LargeContextFallback: largeContextFallback,
		LargeOutputFallback:  largeOutputFallback,
		ErrorFallback:        errorFallback,
		StrongModel:          strongModel,
	}
}

func (m ModelRoleConfig) GetModelId() ModelId {
	if m.BaseModelConfig != nil {
		return m.BaseModelConfig.ModelId
	}

	return m.ModelId
}

func (m ModelRoleConfig) GetBaseModelConfig(authVars map[string]string) *BaseModelConfig {
	if m.BaseModelConfig != nil {
		return m.BaseModelConfig
	}

	foundProvider := m.GetFirstProviderForAuthVars(authVars)
	if foundProvider == nil {
		return m.BaseModelConfig
	}

	c := GetAvailableModel(foundProvider.Provider, m.ModelId).BaseModelConfig

	return &c
}

func (m ModelRoleConfig) GetProviderComposite(authVars map[string]string) string {
	baseModelConfig := m.GetBaseModelConfig(authVars)

	if baseModelConfig == nil {
		return ""
	}

	return baseModelConfig.ToComposite()
}

func (m ModelRoleConfig) GetReservedOutputTokens() int {
	if m.ReservedOutputTokens > 0 {
		return m.ReservedOutputTokens
	}

	sharedBaseConfig := m.GetSharedBaseConfig()
	return sharedBaseConfig.ReservedOutputTokens
}

func (m ModelRoleConfig) GetSharedBaseConfig() *BaseModelShared {
	if m.BaseModelConfig != nil {
		return &m.BaseModelConfig.BaseModelShared
	}

	builtInModel := BuiltInBaseModelsById[m.ModelId]
	if builtInModel == nil {
		return nil
	}

	return &builtInModel.BaseModelShared
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

func (p PlannerRoleConfig) GetMaxConvoTokens() int {
	if p.MaxConvoTokens > 0 {
		return p.MaxConvoTokens
	}

	return p.ModelRoleConfig.GetSharedBaseConfig().DefaultMaxConvoTokens
}

type ModelPackSchema struct {
	Name             string                 `json:"name"`
	Description      string                 `json:"description"`
	Planner          ModelRoleConfigSchema  `json:"planner"`
	Coder            *ModelRoleConfigSchema `json:"coder"`
	PlanSummary      ModelRoleConfigSchema  `json:"planSummary"`
	Builder          ModelRoleConfigSchema  `json:"builder"`
	WholeFileBuilder *ModelRoleConfigSchema `json:"wholeFileBuilder"` // optional, defaults to builder model — access via GetWholeFileBuilder()
	Namer            ModelRoleConfigSchema  `json:"namer"`
	CommitMsg        ModelRoleConfigSchema  `json:"commitMsg"`
	ExecStatus       ModelRoleConfigSchema  `json:"execStatus"`
	Architect        *ModelRoleConfigSchema `json:"contextLoader"`
}

func (m *ModelPackSchema) ToModelPack() ModelPack {
	var (
		coder            *ModelRoleConfig
		wholeFileBuilder *ModelRoleConfig
		architect        *ModelRoleConfig
	)

	if m.Coder != nil {
		c := m.Coder.ToModelRoleConfig()
		coder = &c
	}

	if m.WholeFileBuilder != nil {
		c := m.WholeFileBuilder.ToModelRoleConfig()
		wholeFileBuilder = &c
	}

	if m.Architect != nil {
		c := m.Architect.ToModelRoleConfig()
		architect = &c
	}

	return ModelPack{
		Name:        m.Name,
		Description: m.Description,
		Planner: PlannerRoleConfig{
			ModelRoleConfig: m.Planner.ToModelRoleConfig(),
			PlannerModelConfig: PlannerModelConfig{
				MaxConvoTokens: m.Planner.MaxConvoTokens,
			},
		},
		Coder:            coder,
		PlanSummary:      m.PlanSummary.ToModelRoleConfig(),
		Builder:          m.Builder.ToModelRoleConfig(),
		WholeFileBuilder: wholeFileBuilder,
		Namer:            m.Namer.ToModelRoleConfig(),
		CommitMsg:        m.CommitMsg.ToModelRoleConfig(),
		ExecStatus:       m.ExecStatus.ToModelRoleConfig(),
		Architect:        architect,
	}
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
