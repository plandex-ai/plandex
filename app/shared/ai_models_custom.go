package shared

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

type SchemaUrl string

const (
	SchemaUrlInputConfig     SchemaUrl = "https://plandex.ai/schemas/models-input.schema.json"
	SchemaUrlPlanConfig      SchemaUrl = "https://plandex.ai/schemas/plan-config.schema.json"
	SchemaUrlInlineModelPack SchemaUrl = "https://plandex.ai/schemas/model-pack-inline.schema.json"
)

// Note that none of the custom model structs should have maps anywhere in the hierarchy, since it will break deterministic hashing. Use structs or slices instead.

type CustomModel struct {
	Id          string         `json:"id,omitempty"`
	ModelId     ModelId        `json:"modelId"`
	Publisher   ModelPublisher `json:"publisher"`
	Description string         `json:"description"`

	BaseModelShared

	Providers []BaseModelUsesProvider `json:"providers"`

	CreatedAt *time.Time `json:"createdAt,omitempty"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
}

type CustomProvider struct {
	Id      string `json:"id,omitempty"`
	Name    string `json:"name"`
	BaseUrl string `json:"baseUrl"`

	// for AWS Bedrock models
	HasAWSAuth bool `json:"hasAWSAuth,omitempty"`

	// for local models that don't require auth (ollama, etc.)
	SkipAuth bool `json:"skipAuth,omitempty"`

	ApiKeyEnvVar  string                       `json:"apiKeyEnvVar,omitempty"`
	ExtraAuthVars []ModelProviderExtraAuthVars `json:"extraAuthVars,omitempty"`

	CreatedAt *time.Time `json:"createdAt,omitempty"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
}

type ModelsInput struct {
	CustomModels     []*CustomModel     `json:"models"`
	CustomProviders  []*CustomProvider  `json:"providers,omitempty"`
	CustomModelPacks []*ModelPackSchema `json:"modelPacks"`
}

func (input ModelsInput) FilterUnchanged(existing *ModelsInput) ModelsInput {
	filtered := ModelsInput{}

	existingProvidersById := map[string]*CustomProvider{}
	for _, provider := range existing.CustomProviders {
		existingProvidersById[provider.Name] = provider
	}
	existingModelsById := map[string]*CustomModel{}
	for _, model := range existing.CustomModels {
		existingModelsById[string(model.ModelId)] = model
	}
	existingPacksById := map[string]*ModelPackSchema{}
	for _, pack := range existing.CustomModelPacks {
		existingPacksById[pack.Name] = pack
	}

	for _, model := range input.CustomModels {
		if existingModel, ok := existingModelsById[string(model.ModelId)]; !ok || !modelsEqual(model, existingModel) {
			filtered.CustomModels = append(filtered.CustomModels, model)
		}
	}

	for _, provider := range input.CustomProviders {
		if existingProvider, ok := existingProvidersById[provider.Name]; !ok || !providersEqual(provider, existingProvider) {
			filtered.CustomProviders = append(filtered.CustomProviders, provider)
		}
	}

	for _, pack := range input.CustomModelPacks {
		if existingPack, ok := existingPacksById[pack.Name]; !ok || !packsEqual(pack, existingPack) {
			filtered.CustomModelPacks = append(filtered.CustomModelPacks, pack)
		}
	}

	return filtered
}

func (input ModelsInput) Equals(other ModelsInput) bool {
	left := input.FilterUnchanged(&other)
	right := other.FilterUnchanged(&input)

	return left.IsEmpty() && right.IsEmpty()
}

func (input ModelsInput) CheckNoDuplicates() (bool, string) {
	sawModelIds := map[ModelId]bool{}
	sawProviderNames := map[string]bool{}
	sawPackNames := map[string]bool{}

	builder := strings.Builder{}

	for _, provider := range input.CustomProviders {
		if _, ok := sawProviderNames[provider.Name]; ok {
			builder.WriteString(fmt.Sprintf("• Provider %s is duplicated\n", provider.Name))
		}
		sawProviderNames[provider.Name] = true
	}

	for _, model := range input.CustomModels {
		if _, ok := sawModelIds[model.ModelId]; ok {
			builder.WriteString(fmt.Sprintf("• Model %s is duplicated\n", model.ModelId))
		}
		sawModelIds[model.ModelId] = true
	}

	for _, pack := range input.CustomModelPacks {
		if _, ok := sawPackNames[pack.Name]; ok {
			builder.WriteString(fmt.Sprintf("• Model pack %s is duplicated\n", pack.Name))
		}
		sawPackNames[pack.Name] = true
	}

	res := builder.String()

	return len(res) == 0, res
}

func (input ModelsInput) IsEmpty() bool {
	return len(input.CustomModels) == 0 && len(input.CustomProviders) == 0 && len(input.CustomModelPacks) == 0
}

func modelsEqual(a, b *CustomModel) bool {
	return cmp.Equal(
		a, b,
		cmpopts.EquateEmpty(), // treat nil == empty slice/map
		cmpopts.IgnoreFields(CustomModel{}, "CreatedAt", "UpdatedAt", "Id"),
	)
}

func providersEqual(a, b *CustomProvider) bool {
	return cmp.Equal(
		a,
		b,
		cmpopts.EquateEmpty(),
		cmpopts.IgnoreFields(CustomProvider{}, "CreatedAt", "UpdatedAt", "Id"),
	)
}

func packsEqual(a, b *ModelPackSchema) bool {
	res := cmp.Equal(
		a,
		b,
		cmpopts.EquateEmpty(),
	)
	return res
}

func (s *ModelPackSchema) Equals(other *ModelPackSchema) bool {
	return packsEqual(s, other)
}

func (mp *ModelPack) Equals(other *ModelPack) bool {
	return mp.ToModelPackSchema().Equals(other.ToModelPackSchema())
}

// Hash returns a deterministic hash of the ModelsInput.
// WARNING: This relies on json.Marshal being deterministic for our struct types.
// Do not add map fields to these structs or the hash will become non-deterministic.
func (input ModelsInput) Hash() (string, error) {
	data, err := json.Marshal(input)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

type ClientModelPackSchema struct {
	Name        string `json:"name"`
	Description string `json:"description"`

	ClientModelPackSchemaRoles
}

func (input *ClientModelPackSchema) ToModelPackSchema() *ModelPackSchema {
	return &ModelPackSchema{
		Name:                 input.Name,
		Description:          input.Description,
		ModelPackSchemaRoles: input.ClientModelPackSchemaRoles.ToModelPackSchemaRoles(),
	}
}

func (input *ModelPackSchema) ToClientModelPackSchema() *ClientModelPackSchema {
	return &ClientModelPackSchema{
		Name:                       input.Name,
		Description:                input.Description,
		ClientModelPackSchemaRoles: input.ToClientModelPackSchemaRoles(),
	}
}

type ClientModelsInput struct {
	SchemaUrl SchemaUrl `json:"$schema"`

	CustomModels     []*CustomModel           `json:"models"`
	CustomProviders  []*CustomProvider        `json:"providers,omitempty"`
	CustomModelPacks []*ClientModelPackSchema `json:"modelPacks"`
}

func (input ClientModelsInput) ToModelsInput() ModelsInput {
	modelPacks := []*ModelPackSchema{}
	for _, pack := range input.CustomModelPacks {
		modelPacks = append(modelPacks, pack.ToModelPackSchema())
	}

	return ModelsInput{
		CustomModels:     input.CustomModels,
		CustomProviders:  input.CustomProviders,
		CustomModelPacks: modelPacks,
	}
}

func (input *ClientModelsInput) PrepareUpdate() {
	for _, model := range input.CustomModels {
		model.Id = ""
		model.CreatedAt = nil
		model.UpdatedAt = nil
	}

	for _, provider := range input.CustomProviders {
		provider.Id = ""
		provider.CreatedAt = nil
		provider.UpdatedAt = nil
	}
}

func (input ModelsInput) ToClientModelsInput() ClientModelsInput {
	clientModelPacks := []*ClientModelPackSchema{}
	for _, pack := range input.CustomModelPacks {
		clientModelPacks = append(clientModelPacks, pack.ToClientModelPackSchema())
	}

	return ClientModelsInput{
		SchemaUrl:        SchemaUrlInputConfig,
		CustomModels:     input.CustomModels,
		CustomProviders:  input.CustomProviders,
		CustomModelPacks: clientModelPacks,
	}
}
