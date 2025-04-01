package db

import (
	"fmt"
)

func CreateCustomModel(model *AvailableModel) error {
	query := `INSERT INTO custom_models (org_id, provider, custom_provider, base_url, model_name, model_id, description, max_tokens, api_key_env_var, default_max_convo_tokens, max_output_tokens, reserved_output_tokens, preferred_output_format, has_image_support) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	RETURNING id, created_at, updated_at`

	err := Conn.QueryRow(query, model.OrgId, model.Provider, model.CustomProvider, model.BaseUrl, model.ModelName, model.ModelId, model.Description, model.MaxTokens, model.ApiKeyEnvVar, model.DefaultMaxConvoTokens, model.MaxOutputTokens, model.ReservedOutputTokens, model.PreferredOutputFormat, model.HasImageSupport).Scan(&model.Id, &model.CreatedAt, &model.UpdatedAt)
	if err != nil {
		return fmt.Errorf("error inserting new custom model: %v", err)
	}

	return nil
}

func UpdateCustomModel(model *AvailableModel) error {
	query := `UPDATE custom_models SET provider = $2, custom_provider = $3, base_url = $4, model_name = $5, model_id = $6, description = $7, max_tokens = $8, api_key_env_var = $9, default_max_convo_tokens = $10, max_output_tokens = $11, reserved_output_tokens = $12, preferred_output_format = $13, has_image_support = $14 WHERE id = $1`
	_, err := Conn.Exec(query, model.Id, model.Provider, model.CustomProvider, model.BaseUrl, model.ModelName, model.ModelId, model.Description, model.MaxTokens, model.ApiKeyEnvVar, model.DefaultMaxConvoTokens, model.MaxOutputTokens, model.ReservedOutputTokens, model.PreferredOutputFormat, model.HasImageSupport)
	if err != nil {
		return fmt.Errorf("error updating custom model: %v", err)
	}

	return nil
}

func ListCustomModels(orgId string) ([]*AvailableModel, error) {
	var models []*AvailableModel

	query := `SELECT * FROM custom_models WHERE org_id = $1`

	err := Conn.Select(&models, query, orgId)

	if err != nil {
		return nil, fmt.Errorf("error fetching custom models: %v", err)
	}

	return models, nil
}

func DeleteAvailableModel(modelId string) error {
	query := `DELETE FROM custom_models WHERE id = $1`
	_, err := Conn.Exec(query, modelId)

	if err != nil {
		return fmt.Errorf("error deleting custom model: %v", err)
	}

	return nil
}

func CreateModelPack(ms *ModelPack) error {
	query := `INSERT INTO model_sets (org_id, name, description, planner, coder, plan_summary, builder, whole_file_builder, namer, commit_msg, exec_status, context_loader) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	RETURNING id, created_at`

	err := Conn.QueryRow(query, ms.OrgId, ms.Name, ms.Description, ms.Planner, ms.Coder, ms.PlanSummary, ms.Builder, ms.WholeFileBuilder, ms.Namer, ms.CommitMsg, ms.ExecStatus, ms.Architect).Scan(&ms.Id, &ms.CreatedAt)

	if err != nil {
		return fmt.Errorf("error inserting new model pack: %v", err)
	}

	return nil
}

func UpdateModelPack(ms *ModelPack) error {
	query := `UPDATE model_sets SET name = $2, description = $3, planner = $4, coder = $5, plan_summary = $6, builder = $7, whole_file_builder = $8, namer = $9, commit_msg = $10, exec_status = $11, context_loader = $12 WHERE id = $1`
	_, err := Conn.Exec(query, ms.Id, ms.Name, ms.Description, ms.Planner, ms.Coder, ms.PlanSummary, ms.Builder, ms.WholeFileBuilder, ms.Namer, ms.CommitMsg, ms.ExecStatus, ms.Architect)

	if err != nil {
		return fmt.Errorf("error updating model pack: %v", err)
	}

	return nil
}

func ListModelPacks(orgId string) ([]*ModelPack, error) {
	var modelPacks []*ModelPack

	query := `SELECT * FROM model_sets WHERE org_id = $1`
	err := Conn.Select(&modelPacks, query, orgId)

	if err != nil {
		return nil, fmt.Errorf("error fetching model packs: %v", err)
	}

	return modelPacks, nil
}

func DeleteModelPack(setId string) error {
	query := `DELETE FROM model_sets WHERE id = $1`
	_, err := Conn.Exec(query, setId)

	if err != nil {
		return fmt.Errorf("error deleting model pack: %v", err)
	}

	return nil
}
