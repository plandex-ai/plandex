package shared

type ModelProvider string

const ModelProviderOpenAI ModelProvider = "openai"

type ModelRole string

const (
	ModelRolePlanner     ModelRole = "planner"
	ModelRolePlanSummary ModelRole = "summarizer"
	ModelRoleBuilder     ModelRole = "builder"
	ModelRoleName        ModelRole = "names"
	ModelRoleCommitMsg   ModelRole = "commit-messages"
	ModelRoleExecStatus  ModelRole = "auto-complete"
)

var AllModelRoles = []ModelRole{ModelRolePlanner, ModelRolePlanSummary, ModelRoleBuilder, ModelRoleName, ModelRoleCommitMsg, ModelRoleExecStatus}
var ModelRoleDescriptions = map[ModelRole]string{
	ModelRolePlanner:     "replies to prompts and makes plans",
	ModelRolePlanSummary: "summarizes conversations exceeding max-convo-tokens",
	ModelRoleBuilder:     "builds a plan into file diffs",
	ModelRoleName:        "names plans",
	ModelRoleCommitMsg:   "writes commit messages",
	ModelRoleExecStatus:  "determines whether to auto-continue",
}
var SettingDescriptions = map[string]string{
	"max-convo-tokens":       "max conversation 🪙 before summarization",
	"max-tokens":             "overall 🪙 limit",
	"reserved-output-tokens": "🪙 reserved for model output",
}

var ModelOverridePropsDasherized = []string{"max-convo-tokens", "max-tokens", "reserved-output-tokens"}

func (ps PlanSettings) GetPlannerMaxTokens() int {
	if ps.ModelOverrides.MaxTokens == nil {
		if ps.ModelSet == nil {
			return DefaultModelSet.Planner.BaseModelConfig.MaxTokens
		} else {
			return ps.ModelSet.Planner.BaseModelConfig.MaxTokens
		}
	} else {
		return *ps.ModelOverrides.MaxTokens
	}
}

func (ps PlanSettings) GetPlannerMaxConvoTokens() int {
	if ps.ModelOverrides.MaxConvoTokens == nil {
		if ps.ModelSet == nil {
			return DefaultModelSet.Planner.PlannerModelConfig.MaxConvoTokens
		} else {
			return ps.ModelSet.Planner.PlannerModelConfig.MaxConvoTokens
		}
	} else {
		return *ps.ModelOverrides.MaxConvoTokens
	}
}

func (ps PlanSettings) GetPlannerReservedOutputTokens() int {
	if ps.ModelOverrides.ReservedOutputTokens == nil {
		if ps.ModelSet == nil {
			return DefaultModelSet.Planner.PlannerModelConfig.ReservedOutputTokens
		} else {
			return ps.ModelSet.Planner.PlannerModelConfig.ReservedOutputTokens
		}
	} else {
		return *ps.ModelOverrides.ReservedOutputTokens
	}
}

func (ps PlanSettings) GetPlannerEffectiveMaxTokens() int {
	return ps.GetPlannerMaxTokens() - ps.GetPlannerReservedOutputTokens()
}
