package shared

var FullCompatibility = ModelCompatibility{
	HasImageSupport: true,
}

var RequiredCompatibilityByRole = map[ModelRole]ModelCompatibility{
	ModelRolePlanner:          {},
	ModelRolePlanSummary:      {},
	ModelRoleBuilder:          {},
	ModelRoleName:             {},
	ModelRoleCommitMsg:        {},
	ModelRoleExecStatus:       {},
	ModelRoleArchitect:        {},
	ModelRoleCoder:            {},
	ModelRoleWholeFileBuilder: {},
}

func FilterBuiltInCompatibleModels(models []*BaseModelConfigSchema, role ModelRole) []*BaseModelConfigSchema {
	// required := RequiredCompatibilityByRole[role]
	var compatibleModels []*BaseModelConfigSchema

	for _, model := range models {
		// no compatibility checks are needed in v2, but keeping this here in case compatibility checks are needed in the future

		compatibleModels = append(compatibleModels, model)
	}

	return compatibleModels
}

func FilterCustomCompatibleModels(models []*CustomModel, role ModelRole) []*CustomModel {
	// required := RequiredCompatibilityByRole[role]
	var compatibleModels []*CustomModel

	for _, model := range models {
		// no compatibility checks are needed in v2, but keeping this here in case compatibility checks are needed in the future

		compatibleModels = append(compatibleModels, model)
	}

	return compatibleModels
}
