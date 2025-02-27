package shared

func getPlannerModelConfig(provider ModelProvider, modelId ModelId) PlannerModelConfig {
	return PlannerModelConfig{
		MaxConvoTokens: GetAvailableModel(provider, modelId).DefaultMaxConvoTokens,
	}
}

var DefaultConfigByRole = map[ModelRole]ModelRoleConfig{
	ModelRolePlanner: {
		Temperature: 0.3,
		TopP:        0.3,
	},
	ModelRoleCoder: {
		Temperature: 0.3,
		TopP:        0.3,
	},
	ModelRoleArchitect: {
		Temperature: 0.3,
		TopP:        0.3,
	},
	ModelRolePlanSummary: {
		Temperature: 0.2,
		TopP:        0.2,
	},
	ModelRoleBuilder: {
		Temperature: 0.1,
		TopP:        0.1,
	},
	ModelRoleWholeFileBuilder: {
		Temperature: 0.1,
		TopP:        0.1,
	},
	ModelRoleName: {
		Temperature: 0.8,
		TopP:        0.5,
	},
	ModelRoleCommitMsg: {
		Temperature: 0.8,
		TopP:        0.5,
	},
	ModelRoleExecStatus: {
		Temperature: 0.1,
		TopP:        0.1,
	},
}
