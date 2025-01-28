package shared

type ModelProvider string

const (
	ModelProviderOpenAI ModelProvider = "openai"
	// ModelProviderTogether   ModelProvider = "together" // removing for now to simplify
	ModelProviderOpenRouter ModelProvider = "openrouter"
	ModelProviderCustom     ModelProvider = "custom"
)

var AllModelProviders = []string{
	string(ModelProviderOpenAI),
	string(ModelProviderOpenRouter),
	// string(ModelProviderTogether),
	string(ModelProviderCustom),
}

var BaseUrlByProvider = map[ModelProvider]string{
	ModelProviderOpenAI: OpenAIV1BaseUrl,
	// ModelProviderTogether:   "https://api.together.xyz/v1", // removing for now to simplify
	ModelProviderOpenRouter: "https://openrouter.ai/api/v1",
}

var ApiKeyByProvider = map[ModelProvider]string{
	ModelProviderOpenAI: OpenAIEnvVar,
	// ModelProviderTogether:   "TOGETHER_API_KEY", // removing for now to simplify
	ModelProviderOpenRouter: "OPENROUTER_API_KEY",
}

type ModelRole string

const (
	ModelRolePlanner          ModelRole = "planner"
	ModelRoleCoder            ModelRole = "coder"
	ModelRoleContextLoader    ModelRole = "context-loader"
	ModelRolePlanSummary      ModelRole = "summarizer"
	ModelRoleBuilder          ModelRole = "builder"
	ModelRoleWholeFileBuilder ModelRole = "whole-file-builder"
	ModelRoleName             ModelRole = "names"
	ModelRoleCommitMsg        ModelRole = "commit-messages"
	ModelRoleExecStatus       ModelRole = "auto-continue"
)

var AllModelRoles = []ModelRole{ModelRolePlanner, ModelRoleCoder, ModelRoleContextLoader, ModelRolePlanSummary, ModelRoleBuilder, ModelRoleWholeFileBuilder, ModelRoleName, ModelRoleCommitMsg, ModelRoleExecStatus}
var ModelRoleDescriptions = map[ModelRole]string{
	ModelRolePlanner:          "replies to prompts and makes plans",
	ModelRoleCoder:            "writes code to implement a plan",
	ModelRolePlanSummary:      "summarizes conversations exceeding max-convo-tokens",
	ModelRoleBuilder:          "builds a plan into file diffs",
	ModelRoleWholeFileBuilder: "builds a plan into file diffs by writing the entire file",
	ModelRoleName:             "names plans",
	ModelRoleCommitMsg:        "writes commit messages",
	ModelRoleExecStatus:       "determines whether to auto-continue",
	ModelRoleContextLoader:    "decides what context to load using codebase map",
}
var SettingDescriptions = map[string]string{
	"max-convo-tokens":       "max conversation 🪙 before summarization",
	"max-tokens":             "overall 🪙 limit",
	"reserved-output-tokens": "🪙 reserved for model output",
}

var ModelOverridePropsDasherized = []string{"max-convo-tokens", "max-tokens", "reserved-output-tokens"}

func (ps PlanSettings) GetPlannerMaxTokens() int {
	if ps.ModelOverrides.MaxTokens == nil {
		if ps.ModelPack == nil {
			return DefaultModelPack.Planner.BaseModelConfig.MaxTokens
		} else {
			return ps.ModelPack.Planner.BaseModelConfig.MaxTokens
		}
	} else {
		return *ps.ModelOverrides.MaxTokens
	}
}

func (ps PlanSettings) GetPlannerMaxConvoTokens() int {
	if ps.ModelOverrides.MaxConvoTokens == nil {
		if ps.ModelPack == nil {
			return DefaultModelPack.Planner.PlannerModelConfig.MaxConvoTokens
		} else {
			return ps.ModelPack.Planner.PlannerModelConfig.MaxConvoTokens
		}
	} else {
		return *ps.ModelOverrides.MaxConvoTokens
	}
}

func (ps PlanSettings) GetPlannerEffectiveMaxTokens() int {
	return ps.GetPlannerMaxTokens() - ps.ModelPack.Planner.GetReservedOutputTokens()
}

func (ps PlanSettings) GetRequiredEnvVars() map[string]bool {
	envVars := map[string]bool{}

	ms := ps.ModelPack
	if ms == nil {
		ms = DefaultModelPack
	}

	envVars[ms.Planner.BaseModelConfig.ApiKeyEnvVar] = true
	envVars[ms.Builder.BaseModelConfig.ApiKeyEnvVar] = true
	envVars[ms.WholeFileBuilder.BaseModelConfig.ApiKeyEnvVar] = true
	envVars[ms.PlanSummary.BaseModelConfig.ApiKeyEnvVar] = true
	envVars[ms.Namer.BaseModelConfig.ApiKeyEnvVar] = true
	envVars[ms.CommitMsg.BaseModelConfig.ApiKeyEnvVar] = true
	envVars[ms.ExecStatus.BaseModelConfig.ApiKeyEnvVar] = true
	envVars[ms.ContextLoader.BaseModelConfig.ApiKeyEnvVar] = true
	envVars[ms.Coder.BaseModelConfig.ApiKeyEnvVar] = true

	// for backward compatibility with <= 0.8.4 server versions
	if len(envVars) == 0 {
		envVars["OPENAI_API_KEY"] = true
	}

	return envVars
}
