package shared

var fullCompatibility = ModelCompatibility{
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

func FilterCompatibleModels(models []*AvailableModel, role ModelRole) []*AvailableModel {
	// required := RequiredCompatibilityByRole[role]
	var compatibleModels []*AvailableModel

	for _, model := range models {
		// no compatibility checks are needed in v2, but keeping this here in case compatibility checks are needed in the future

		compatibleModels = append(compatibleModels, model)
	}

	return compatibleModels
}
