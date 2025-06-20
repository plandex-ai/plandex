package shared

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

type PlanSettings struct {
	ModelPackName    string       `json:"modelPackName"`
	ModelPack        *ModelPack   `json:"modelPack"`
	CustomModelPacks []*ModelPack `json:"customModelPacks"`
	IsCloud          bool         `json:"isCloud"`
	Configured       bool         `json:"configured"`
	UpdatedAt        time.Time    `json:"updatedAt"`
}

func (p *PlanSettings) Configure(customModelPacks []*ModelPack, isCloud bool) {
	p.CustomModelPacks = customModelPacks
	p.IsCloud = isCloud
	p.Configured = true
}

func (p PlanSettings) GetModelPack() *ModelPack {
	if !p.Configured {
		panic("PlanSettings not configured")
	}

	customModelPacks := p.CustomModelPacks
	isCloud := p.IsCloud

	fillDefault := true // seems best to make this the default behavior, but keeping the switch just in case

	if p.ModelPack != nil {
		return p.ModelPack
	}

	for _, builtInModelPack := range BuiltInModelPacks {
		if isCloud && builtInModelPack.LocalProvider != "" {
			continue
		}

		if builtInModelPack.Name == p.ModelPackName {
			return builtInModelPack
		}
	}

	for _, customModelPack := range customModelPacks {
		if customModelPack.Name == p.ModelPackName {
			return customModelPack
		}
	}

	if fillDefault {
		return DefaultModelPack
	}

	return nil
}

func (p *PlanSettings) SetModelPackByName(modelPackName string) {
	p.ModelPackName = modelPackName
	p.ModelPack = nil
}

func (p *PlanSettings) SetCustomModelPack(modelPack *ModelPack) {
	p.ModelPackName = ""
	p.ModelPack = modelPack
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

func (ps PlanSettings) GetPlannerMaxTokens() int {
	modelPack := ps.GetModelPack()
	planner := modelPack.Planner
	fallback := planner.GetFinalLargeContextFallback()
	baseConfig := fallback.GetSharedBaseConfig()
	return baseConfig.MaxTokens
}

func (ps PlanSettings) GetPlannerMaxReservedOutputTokens() int {
	modelPack := ps.GetModelPack()
	planner := modelPack.Planner
	return planner.GetFinalLargeContextFallback().GetReservedOutputTokens()
}

func (ps PlanSettings) GetArchitectMaxTokens() int {
	modelPack := ps.GetModelPack()
	architect := modelPack.GetArchitect()
	fallback := architect.GetFinalLargeContextFallback()
	return fallback.GetSharedBaseConfig().MaxTokens
}

func (ps PlanSettings) GetArchitectMaxReservedOutputTokens() int {
	modelPack := ps.GetModelPack()
	architect := modelPack.GetArchitect()
	fallback := architect.GetFinalLargeContextFallback()
	return fallback.GetReservedOutputTokens()
}

func (ps PlanSettings) GetCoderMaxTokens() int {
	modelPack := ps.GetModelPack()
	coder := modelPack.GetCoder()
	fallback := coder.GetFinalLargeContextFallback()
	return fallback.GetSharedBaseConfig().MaxTokens
}

func (ps PlanSettings) GetCoderMaxReservedOutputTokens() int {
	modelPack := ps.GetModelPack()
	coder := modelPack.GetCoder()
	fallback := coder.GetFinalLargeContextFallback()
	return fallback.GetReservedOutputTokens()
}

func (ps PlanSettings) GetWholeFileBuilderMaxTokens() int {
	modelPack := ps.GetModelPack()
	builder := modelPack.GetWholeFileBuilder()
	fallback := builder.GetFinalLargeContextFallback()
	return fallback.GetSharedBaseConfig().MaxTokens
}

func (ps PlanSettings) GetWholeFileBuilderMaxReservedOutputTokens() int {
	modelPack := ps.GetModelPack()
	builder := modelPack.GetWholeFileBuilder()
	fallback := builder.GetFinalLargeOutputFallback()
	return fallback.GetReservedOutputTokens()
}

func (ps PlanSettings) GetPlannerMaxConvoTokens() int {
	modelPack := ps.GetModelPack()

	// for max convo tokens, we use the planner's default max convo tokens, *not* the fallback, so that we don't end up switching to the fallback just based on the conversation length
	planner := modelPack.Planner
	if planner.MaxConvoTokens != 0 {
		return planner.MaxConvoTokens
	}

	return planner.GetSharedBaseConfig().DefaultMaxConvoTokens
}

func (ps PlanSettings) GetPlannerEffectiveMaxTokens() int {
	maxPlannerTokens := ps.GetPlannerMaxTokens()
	maxReservedOutputTokens := ps.GetPlannerMaxReservedOutputTokens()

	return maxPlannerTokens - maxReservedOutputTokens
}

func (ps PlanSettings) GetArchitectEffectiveMaxTokens() int {
	maxArchitectTokens := ps.GetArchitectMaxTokens()
	maxReservedOutputTokens := ps.GetArchitectMaxReservedOutputTokens()

	return maxArchitectTokens - maxReservedOutputTokens
}

func (ps PlanSettings) GetCoderEffectiveMaxTokens() int {
	maxCoderTokens := ps.GetCoderMaxTokens()
	maxReservedOutputTokens := ps.GetCoderMaxReservedOutputTokens()

	return maxCoderTokens - maxReservedOutputTokens
}

func (ps PlanSettings) GetWholeFileBuilderEffectiveMaxTokens() int {
	maxWholeFileBuilderTokens := ps.GetWholeFileBuilderMaxTokens()
	maxReservedOutputTokens := ps.GetWholeFileBuilderMaxReservedOutputTokens()

	return maxWholeFileBuilderTokens - maxReservedOutputTokens
}

func (ps PlanSettings) GetModelProviderOptions() ModelProviderOptions {
	opts := ModelProviderOptions{}

	ms := ps.GetModelPack()
	if ms == nil {
		ms = DefaultModelPack
	}

	opts = opts.Condense(
		ms.Planner.GetModelProviderOptions(),
		ms.Builder.GetModelProviderOptions(),
		ms.PlanSummary.GetModelProviderOptions(),
		ms.Namer.GetModelProviderOptions(),
		ms.CommitMsg.GetModelProviderOptions(),
		ms.ExecStatus.GetModelProviderOptions(),
		// optional roles
		getOptionalModelProviderOptions(ms.WholeFileBuilder),
		getOptionalModelProviderOptions(ms.Architect),
		getOptionalModelProviderOptions(ms.Coder),
	)

	return opts
}

func (ps *PlanSettings) Equals(other *PlanSettings) bool {
	return ps.GetModelPack().Equals(other.GetModelPack())
}

func (ps PlanSettings) ForCompare() PlanSettings {
	ps.UpdatedAt = time.Time{}
	ps.CustomModelPacks = nil
	ps.IsCloud = false
	ps.Configured = false
	return ps
}

func (ps PlanSettings) DeepCopy() (*PlanSettings, error) {
	bytes, err := json.Marshal(ps)
	if err != nil {
		return nil, fmt.Errorf("error marshalling plan settings: %v", err)
	}
	var copy PlanSettings
	err = json.Unmarshal(bytes, &copy)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling plan settings: %v", err)
	}
	return &copy, nil
}

func getOptionalModelProviderOptions(cfg *ModelRoleConfig) ModelProviderOptions {
	if cfg == nil {
		return ModelProviderOptions{}
	}
	return cfg.GetModelProviderOptions()
}
