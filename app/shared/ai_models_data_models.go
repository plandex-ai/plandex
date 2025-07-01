package shared

import (
	"crypto/sha256"
	"database/sql/driver"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
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
	DefaultMaxConvoTokens       int               `json:"defaultMaxConvoTokens"`
	MaxTokens                   int               `json:"maxTokens"`
	MaxOutputTokens             int               `json:"maxOutputTokens"`
	ReservedOutputTokens        int               `json:"reservedOutputTokens"`
	PreferredOutputFormat       ModelOutputFormat `json:"preferredOutputFormat"`
	SystemPromptDisabled        bool              `json:"systemPromptDisabled,omitempty"`
	RoleParamsDisabled          bool              `json:"roleParamsDisabled,omitempty"`
	StopDisabled                bool              `json:"stopDisabled,omitempty"`
	PredictedOutputEnabled      bool              `json:"predictedOutputEnabled,omitempty"`
	ReasoningEffortEnabled      bool              `json:"reasoningEffortEnabled,omitempty"`
	ReasoningEffort             ReasoningEffort   `json:"reasoningEffort,omitempty"`
	IncludeReasoning            bool              `json:"includeReasoning,omitempty"`
	HideReasoning               bool              `json:"hideReasoning,omitempty"`
	ReasoningBudget             int               `json:"reasoningBudget,omitempty"`
	SupportsCacheControl        bool              `json:"supportsCacheControl,omitempty"`
	SingleMessageNoSystemPrompt bool              `json:"singleMessageNoSystemPrompt,omitempty"`
	TokenEstimatePaddingPct     float64           `json:"tokenEstimatePaddingPct,omitempty"`
	ModelCompatibility
}

type BaseModelProviderConfig struct {
	ModelProviderConfigSchema
	ModelName ModelName `json:"modelName"`
}

type BaseModelConfig struct {
	ModelTag  ModelTag       `json:"modelTag"`
	ModelId   ModelId        `json:"modelId"`
	Publisher ModelPublisher `json:"publisher,omitempty"`
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
	IsDefaultVariant         bool                     `json:"isDefaultVariant"`
}

func (b *BaseModelConfigSchema) IsLocalOnly() bool {
	if len(b.Providers) == 0 {
		return false
	}

	for _, provider := range b.Providers {
		builtIn, ok := BuiltInModelProviderConfigs[provider.Provider]
		if !ok {
			// has a custom provider—assume not local only
			return false
		}
		if !builtIn.LocalOnly {
			// has a built-in provider that is not local only
			return false
		}
	}

	return true
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
			CumulativeOverrides      BaseModelShared
			RequiresVariantOverrides []string
		}

		addBaseVariant := func(params variantParams) {
			if params.BaseVariant == nil {
				addBase()
				return
			}
			baseDescription := params.BaseVariant.Description
			baseId := params.BaseId
			mergedOverrides := params.CumulativeOverrides

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

					var modelId ModelId
					if variant.IsDefaultVariant {
						modelId = baseId
					} else {
						modelId = ModelId(strings.Join([]string{string(baseId), string(variant.VariantTag)}, "-"))
					}

					description := strings.Join([]string{baseDescription, variant.Description}, " ")

					merged := Merge(baseParams.CumulativeOverrides, variant.Overrides)

					if len(variant.Variants) > 0 {
						addVariants(variant.Variants, variantParams{
							BaseVariant:              &variant,
							BaseId:                   modelId,
							BaseDescription:          description,
							CumulativeOverrides:      merged,
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
				CumulativeOverrides:      b.BaseModelShared,
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
	if m.Provider != "" && m.Provider != ModelProviderOpenAI {
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

	LocalProvider ModelProvider `json:"localProvider,omitempty"`
}

type ModelRoleModelConfig struct {
	Provider       ModelProvider `json:"provider"`
	CustomProvider *string       `json:"customProvider,omitempty"`
	ModelTag       ModelTag      `json:"modelTag"`
}

type ModelRoleConfigSchema struct {
	ModelId ModelId `json:"modelId"`

	Temperature          *float32 `json:"temperature,omitempty"`
	TopP                 *float32 `json:"topP,omitempty"`
	ReservedOutputTokens *int     `json:"reservedOutputTokens,omitempty"`
	MaxConvoTokens       *int     `json:"maxConvoTokens,omitempty"`

	LargeContextFallback *ModelRoleConfigSchema `json:"largeContextFallback,omitempty"`
	LargeOutputFallback  *ModelRoleConfigSchema `json:"largeOutputFallback,omitempty"`
	ErrorFallback        *ModelRoleConfigSchema `json:"errorFallback,omitempty"`
	StrongModel          *ModelRoleConfigSchema `json:"strongModel,omitempty"`
}

// ToClientVal returns either:
//   - string  – when the value (or any nested fallback) is just a bare role
//   - map[string]any – when additional fields are set, with all fallbacks
//     processed recursively.
func (m *ModelRoleConfigSchema) ToClientVal() RoleJSON {
	if m == nil {
		return nil
	}
	if m.bareRole() {
		return string(m.ModelId)
	}

	out := map[string]any{
		"modelId": string(m.ModelId),
	}

	// simple optional scalars
	if m.Temperature != nil {
		out["temperature"] = *m.Temperature
	}
	if m.TopP != nil {
		out["topP"] = *m.TopP
	}
	if m.ReservedOutputTokens != nil {
		out["reservedOutputTokens"] = *m.ReservedOutputTokens
	}
	if m.MaxConvoTokens != nil {
		out["maxConvoTokens"] = *m.MaxConvoTokens
	}

	// recurse on each fallback, collapsing to string when bare
	if m.LargeContextFallback != nil {
		out["largeContextFallback"] = m.LargeContextFallback.ToClientVal()
	}
	if m.LargeOutputFallback != nil {
		out["largeOutputFallback"] = m.LargeOutputFallback.ToClientVal()
	}
	if m.ErrorFallback != nil {
		out["errorFallback"] = m.ErrorFallback.ToClientVal()
	}
	if m.StrongModel != nil {
		out["strongModel"] = m.StrongModel.ToClientVal()
	}

	return out
}

// bareRole returns true if *every* field except ModelId is nil / zero.
func (m *ModelRoleConfigSchema) bareRole() bool {
	if m == nil {
		return true
	}

	v := reflect.ValueOf(*m)
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		f := t.Field(i)
		if f.Name == "ModelId" { // skip the sentinel field
			continue
		}

		fv := v.Field(i)

		switch fv.Kind() {
		case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice:
			if !fv.IsNil() {
				return false
			}
		default:
			if !fv.IsZero() {
				return false
			}
		}
	}
	return true
}

func (m *ModelRoleConfigSchema) AllModelIds() []ModelId {
	ids := []ModelId{}

	if m.ModelId != "" {
		ids = append(ids, m.ModelId)
	}

	if m.LargeContextFallback != nil {
		ids = append(ids, m.LargeContextFallback.AllModelIds()...)
	}

	if m.LargeOutputFallback != nil {
		ids = append(ids, m.LargeOutputFallback.AllModelIds()...)
	}

	if m.ErrorFallback != nil {
		ids = append(ids, m.ErrorFallback.AllModelIds()...)
	}

	if m.StrongModel != nil {
		ids = append(ids, m.StrongModel.AllModelIds()...)
	}

	return ids
}

func (m *ModelRoleConfigSchema) ToModelRoleConfig(role ModelRole) ModelRoleConfig {
	return m.toModelRoleConfig(role)
}

func (m *ModelRoleConfigSchema) toModelRoleConfig(role ModelRole) ModelRoleConfig {
	var largeContextFallback *ModelRoleConfig
	if m.LargeContextFallback != nil {
		c := m.LargeContextFallback.ToModelRoleConfig(role)
		largeContextFallback = &c
	}
	var largeOutputFallback *ModelRoleConfig
	if m.LargeOutputFallback != nil {
		c := m.LargeOutputFallback.ToModelRoleConfig(role)
		largeOutputFallback = &c
	}
	var errorFallback *ModelRoleConfig
	if m.ErrorFallback != nil {
		c := m.ErrorFallback.ToModelRoleConfig(role)
		errorFallback = &c
	}
	var strongModel *ModelRoleConfig
	if m.StrongModel != nil {
		c := m.StrongModel.ToModelRoleConfig(role)
		strongModel = &c
	}

	temperature := m.Temperature
	topP := m.TopP

	config := DefaultConfigByRole[role]

	if temperature == nil {
		temperature = &config.Temperature
	}
	if topP == nil {
		topP = &config.TopP
	}

	var reservedOutputTokens int
	if m.ReservedOutputTokens != nil {
		reservedOutputTokens = *m.ReservedOutputTokens
	}

	return ModelRoleConfig{
		Role: role,

		ModelId: m.ModelId,

		Temperature:          *temperature,
		TopP:                 *topP,
		ReservedOutputTokens: reservedOutputTokens,

		LargeContextFallback: largeContextFallback,
		LargeOutputFallback:  largeOutputFallback,
		ErrorFallback:        errorFallback,
		StrongModel:          strongModel,
	}
}

func (m *ModelRoleConfigSchema) MarshalJSON() ([]byte, error) {
	if m == nil {
		return []byte("null"), nil
	}
	if m.bareRole() {
		if m.ModelId == "" {
			return []byte("null"), nil
		}

		return json.Marshal(string(m.ModelId)) // compact form
	}
	type alias ModelRoleConfigSchema
	return json.Marshal((*alias)(m)) // full object
}

func (m *ModelRoleConfigSchema) UnmarshalJSON(data []byte) error {
	// attempt the short string first
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*m = ModelRoleConfigSchema{ModelId: ModelId(s)}
		return nil
	}
	// fallback to full object
	type alias ModelRoleConfigSchema
	return json.Unmarshal(data, (*alias)(m))
}

func (m *ModelRoleConfig) ToModelRoleConfigSchema() ModelRoleConfigSchema {
	var largeContextFallback *ModelRoleConfigSchema
	if m.LargeContextFallback != nil {
		c := m.LargeContextFallback.ToModelRoleConfigSchema()
		largeContextFallback = &c
	}
	var largeOutputFallback *ModelRoleConfigSchema
	if m.LargeOutputFallback != nil {
		c := m.LargeOutputFallback.ToModelRoleConfigSchema()
		largeOutputFallback = &c
	}
	var errorFallback *ModelRoleConfigSchema
	if m.ErrorFallback != nil {
		c := m.ErrorFallback.ToModelRoleConfigSchema()
		errorFallback = &c
	}
	var strongModel *ModelRoleConfigSchema
	if m.StrongModel != nil {
		c := m.StrongModel.ToModelRoleConfigSchema()
		strongModel = &c
	}

	defaultConfig := DefaultConfigByRole[m.Role]

	var temperature *float32
	var topP *float32
	var reservedOutputTokens *int

	if m.Temperature != defaultConfig.Temperature {
		temperature = &m.Temperature
	}
	if m.TopP != defaultConfig.TopP {
		topP = &m.TopP
	}

	if m.ReservedOutputTokens != 0 {
		reservedOutputTokens = &m.ReservedOutputTokens
	}

	return ModelRoleConfigSchema{
		ModelId:              m.GetModelId(),
		Temperature:          temperature,
		TopP:                 topP,
		ReservedOutputTokens: reservedOutputTokens,
		LargeContextFallback: largeContextFallback,
		LargeOutputFallback:  largeOutputFallback,
		ErrorFallback:        errorFallback,
		StrongModel:          strongModel,
	}
}

func (p PlannerRoleConfig) ToModelRoleConfigSchema() ModelRoleConfigSchema {
	s := p.ModelRoleConfig.ToModelRoleConfigSchema()

	var maxConvoTokens *int
	if p.MaxConvoTokens != 0 {
		maxConvoTokens = &p.MaxConvoTokens
	}

	s.MaxConvoTokens = maxConvoTokens
	return s
}

func (m ModelRoleConfig) GetModelId() ModelId {
	if m.BaseModelConfig != nil {
		return m.BaseModelConfig.ModelId
	}

	return m.ModelId
}

func (m ModelRoleConfig) GetBaseModelConfig(authVars map[string]string, settings *PlanSettings) *BaseModelConfig {
	foundProvider := m.GetFirstProviderForAuthVars(authVars, settings)
	if foundProvider == nil {
		return nil
	}

	return m.GetBaseModelConfigForProvider(authVars, settings, foundProvider)
}

func (m ModelRoleConfig) GetBaseModelConfigForProvider(authVars map[string]string, settings *PlanSettings, providerSchema *ModelProviderConfigSchema) *BaseModelConfig {
	if m.BaseModelConfig != nil {
		return m.BaseModelConfig
	}

	availableModel := GetAvailableModel(providerSchema.Provider, m.ModelId)
	if availableModel != nil {
		c := availableModel.BaseModelConfig
		return &c
	}

	var customModel *CustomModel
	if settings != nil {
		customModel = settings.CustomModelsById[m.ModelId]
	}
	if customModel != nil {
		c := customModel.ToBaseModelConfigForProvider(authVars, settings, providerSchema)
		return c
	}

	return nil
}

func (m ModelRoleConfig) GetProviderComposite(authVars map[string]string, settings *PlanSettings) string {
	baseModelConfig := m.GetBaseModelConfig(authVars, settings)

	if baseModelConfig == nil {
		return ""
	}

	return baseModelConfig.ToComposite()
}

func (m ModelRoleConfig) GetReservedOutputTokens(customModelsById map[ModelId]*CustomModel) int {
	if m.ReservedOutputTokens > 0 {
		return m.ReservedOutputTokens
	}

	sharedBaseConfig := m.GetSharedBaseConfigWithCustomModels(customModelsById)
	return sharedBaseConfig.ReservedOutputTokens
}

func (m ModelRoleConfig) GetSharedBaseConfig(settings *PlanSettings) *BaseModelShared {
	return m.GetSharedBaseConfigWithCustomModels(settings.CustomModelsById)
}

func (m ModelRoleConfig) GetSharedBaseConfigWithCustomModels(customModels map[ModelId]*CustomModel) *BaseModelShared {
	if m.BaseModelConfig != nil {
		return &m.BaseModelConfig.BaseModelShared
	}

	builtInModel := BuiltInBaseModelsById[m.ModelId]
	if builtInModel != nil {
		return &builtInModel.BaseModelShared
	}

	customModel := customModels[m.ModelId]
	if customModel != nil {
		return &customModel.BaseModelShared
	}

	return nil
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

func (p PlannerRoleConfig) GetMaxConvoTokens(settings *PlanSettings) int {
	if p.MaxConvoTokens > 0 {
		return p.MaxConvoTokens
	}

	return p.ModelRoleConfig.GetSharedBaseConfig(settings).DefaultMaxConvoTokens
}

type RoleJSON any

type ClientModelPackSchemaRoles struct {
	SchemaUrl SchemaUrl `json:"$schema,omitempty"`

	LocalProvider ModelProvider `json:"localProvider,omitempty"`

	// in the JSON, these can either be a role as a string or a ModelRoleConfigSchema object for more complex config
	Planner          RoleJSON `json:"planner"`
	Architect        RoleJSON `json:"architect,omitempty"`
	Coder            RoleJSON `json:"coder,omitempty"`
	PlanSummary      RoleJSON `json:"summarizer"`
	Builder          RoleJSON `json:"builder"`
	WholeFileBuilder RoleJSON `json:"wholeFileBuilder,omitempty"`
	Namer            RoleJSON `json:"names"`
	CommitMsg        RoleJSON `json:"commitMessages"`
	ExecStatus       RoleJSON `json:"autoContinue"`
}

func (c *ClientModelPackSchemaRoles) ToModelPackSchemaRoles() ModelPackSchemaRoles {
	res := ModelPackSchemaRoles{
		LocalProvider: c.LocalProvider,
	}

	var convertField func(field interface{}) *ModelRoleConfigSchema
	convertField = func(field interface{}) *ModelRoleConfigSchema {
		if field == nil {
			return nil
		}

		switch v := field.(type) {
		case string:
			// It's a string, handle accordingly
			return &ModelRoleConfigSchema{
				ModelId: ModelId(v),
			}
		case map[string]any:
			// re-marshal then unmarshal into the right struct
			b, _ := json.Marshal(v)
			var m ModelRoleConfigSchema
			if err := json.Unmarshal(b, &m); err != nil {
				return nil
			}

			// Now handle the fallback fields recursively
			if fallback, ok := v["largeContextFallback"]; ok && fallback != nil {
				m.LargeContextFallback = convertField(fallback)
			}
			if fallback, ok := v["largeOutputFallback"]; ok && fallback != nil {
				m.LargeOutputFallback = convertField(fallback)
			}
			if fallback, ok := v["errorFallback"]; ok && fallback != nil {
				m.ErrorFallback = convertField(fallback)
			}
			if fallback, ok := v["strongModel"]; ok && fallback != nil {
				m.StrongModel = convertField(fallback)
			}

			return &m
		default:
			// Handle unexpected type - you might want to log or panic
			// Or try to convert from a map if it's coming from JSON
			return nil
		}
	}

	// Convert each field
	res.Planner = *convertField(c.Planner)
	if c.Coder != nil {
		converted := convertField(c.Coder)
		res.Coder = converted
	}
	res.PlanSummary = *convertField(c.PlanSummary)
	res.Builder = *convertField(c.Builder)
	if c.WholeFileBuilder != nil {
		converted := convertField(c.WholeFileBuilder)
		res.WholeFileBuilder = converted
	}
	res.Namer = *convertField(c.Namer)
	res.CommitMsg = *convertField(c.CommitMsg)
	res.ExecStatus = *convertField(c.ExecStatus)
	if c.Architect != nil {
		converted := convertField(c.Architect)
		res.Architect = converted
	}

	return res
}

type ModelPackSchemaRoles struct {
	LocalProvider    ModelProvider          `json:"localProvider,omitempty"`
	Planner          ModelRoleConfigSchema  `json:"planner"`
	Coder            *ModelRoleConfigSchema `json:"coder,omitempty"`
	PlanSummary      ModelRoleConfigSchema  `json:"planSummary"`
	Builder          ModelRoleConfigSchema  `json:"builder"`
	WholeFileBuilder *ModelRoleConfigSchema `json:"wholeFileBuilder,omitempty"` // optional, defaults to builder model — access via GetWholeFileBuilder()
	Namer            ModelRoleConfigSchema  `json:"namer"`
	CommitMsg        ModelRoleConfigSchema  `json:"commitMsg"`
	ExecStatus       ModelRoleConfigSchema  `json:"execStatus"`
	Architect        *ModelRoleConfigSchema `json:"contextLoader,omitempty"`
}

func (m *ModelPackSchemaRoles) ToClientModelPackSchemaRoles() ClientModelPackSchemaRoles {
	res := ClientModelPackSchemaRoles{
		SchemaUrl:     SchemaUrlInlineModelPack,
		LocalProvider: m.LocalProvider,
	}

	res.Planner = m.Planner.ToClientVal()
	if m.Coder != nil {
		val := m.Coder.ToClientVal()
		res.Coder = &val
	}
	res.PlanSummary = m.PlanSummary.ToClientVal()
	res.Builder = m.Builder.ToClientVal()
	if m.WholeFileBuilder != nil {
		val := m.WholeFileBuilder.ToClientVal()
		res.WholeFileBuilder = &val
	}
	res.Namer = m.Namer.ToClientVal()
	res.CommitMsg = m.CommitMsg.ToClientVal()
	res.ExecStatus = m.ExecStatus.ToClientVal()
	if m.Architect != nil {
		val := m.Architect.ToClientVal()
		res.Architect = &val
	}

	return res
}

type ModelPackSchema struct {
	Name        string `json:"name"`
	Description string `json:"description"`

	ModelPackSchemaRoles
}

func (m *ModelPackSchema) AllModelIds() []ModelId {
	ids := []ModelId{}

	ids = append(ids, m.Planner.AllModelIds()...)

	if m.Coder != nil {
		ids = append(ids, m.Coder.AllModelIds()...)
	}

	ids = append(ids, m.PlanSummary.AllModelIds()...)
	ids = append(ids, m.Builder.AllModelIds()...)

	if m.WholeFileBuilder != nil {
		ids = append(ids, m.WholeFileBuilder.AllModelIds()...)
	}

	ids = append(ids, m.Namer.AllModelIds()...)
	ids = append(ids, m.CommitMsg.AllModelIds()...)
	ids = append(ids, m.ExecStatus.AllModelIds()...)

	if m.Architect != nil {
		ids = append(ids, m.Architect.AllModelIds()...)
	}

	return ids
}

func (m *ModelPackSchema) ToModelPack() ModelPack {
	var (
		coder            *ModelRoleConfig
		wholeFileBuilder *ModelRoleConfig
		architect        *ModelRoleConfig
	)

	if m.Coder != nil {
		c := m.Coder.ToModelRoleConfig(ModelRoleCoder)
		coder = &c
	}

	if m.WholeFileBuilder != nil {
		c := m.WholeFileBuilder.ToModelRoleConfig(ModelRoleWholeFileBuilder)
		wholeFileBuilder = &c
	}

	if m.Architect != nil {
		c := m.Architect.ToModelRoleConfig(ModelRoleArchitect)
		architect = &c
	}

	var maxConvoTokens int
	if m.Planner.MaxConvoTokens != nil {
		maxConvoTokens = *m.Planner.MaxConvoTokens
	}

	return ModelPack{
		Name:          m.Name,
		Description:   m.Description,
		LocalProvider: m.LocalProvider,
		Planner: PlannerRoleConfig{
			ModelRoleConfig: m.Planner.ToModelRoleConfig(ModelRolePlanner),
			PlannerModelConfig: PlannerModelConfig{
				MaxConvoTokens: maxConvoTokens,
			},
		},
		Coder:            coder,
		PlanSummary:      m.PlanSummary.ToModelRoleConfig(ModelRolePlanSummary),
		Builder:          m.Builder.ToModelRoleConfig(ModelRoleBuilder),
		WholeFileBuilder: wholeFileBuilder,
		Namer:            m.Namer.ToModelRoleConfig(ModelRoleName),
		CommitMsg:        m.CommitMsg.ToModelRoleConfig(ModelRoleCommitMsg),
		ExecStatus:       m.ExecStatus.ToModelRoleConfig(ModelRoleExecStatus),
		Architect:        architect,
	}
}

type ModelPack struct {
	Id               string            `json:"id"`
	Name             string            `json:"name"`
	LocalProvider    ModelProvider     `json:"localProvider,omitempty"`
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

func (m *ModelPack) ToModelPackSchema() *ModelPackSchema {
	var coder *ModelRoleConfigSchema
	if m.Coder != nil {
		c := m.Coder.ToModelRoleConfigSchema()
		coder = &c
	}
	var wholeFileBuilder *ModelRoleConfigSchema
	if m.WholeFileBuilder != nil {
		c := m.WholeFileBuilder.ToModelRoleConfigSchema()
		wholeFileBuilder = &c
	}
	var architect *ModelRoleConfigSchema
	if m.Architect != nil {
		c := m.Architect.ToModelRoleConfigSchema()
		architect = &c
	}

	return &ModelPackSchema{
		Name:        m.Name,
		Description: m.Description,
		ModelPackSchemaRoles: ModelPackSchemaRoles{
			LocalProvider:    m.LocalProvider,
			Planner:          m.Planner.ToModelRoleConfigSchema(),
			Coder:            coder,
			Architect:        architect,
			PlanSummary:      m.PlanSummary.ToModelRoleConfigSchema(),
			Builder:          m.Builder.ToModelRoleConfigSchema(),
			WholeFileBuilder: wholeFileBuilder,
			Namer:            m.Namer.ToModelRoleConfigSchema(),
			CommitMsg:        m.CommitMsg.ToModelRoleConfigSchema(),
			ExecStatus:       m.ExecStatus.ToModelRoleConfigSchema(),
		},
	}
}

func (m ModelPackSchemaRoles) Hash() (string, error) {
	bytes, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(bytes)
	return hex.EncodeToString(hash[:]), nil
}
