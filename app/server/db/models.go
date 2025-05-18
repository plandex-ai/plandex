package db

import (
	"database/sql"
	"fmt"
)

func CreateCustomModel(model *CustomModel) error {
	query := `INSERT INTO custom_models (
		org_id,
		model_id,
		publisher,
		description,
		max_tokens,
		default_max_convo_tokens,
		max_output_tokens,
		reserved_output_tokens,
		has_image_support,
		preferred_output_format,
		system_prompt_disabled,
		role_params_disabled,
		stop_disabled,
		predicted_output_enabled,
		reasoning_effort_enabled,
		reasoning_effort,
		include_reasoning,
		reasoning_budget,
		supports_cache_control,
		single_message_no_system_prompt,
		token_estimate_padding_pct,
		providers
	) VALUES (
		$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,
		$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22
	)
	RETURNING id, created_at, updated_at`

	return Conn.QueryRow(
		query,
		model.OrgId,
		model.ModelId,
		model.Publisher,
		model.Description,
		model.MaxTokens,
		model.DefaultMaxConvoTokens,
		model.MaxOutputTokens,
		model.ReservedOutputTokens,
		model.HasImageSupport,
		model.PreferredOutputFormat,
		model.SystemPromptDisabled,
		model.RoleParamsDisabled,
		model.StopDisabled,
		model.PredictedOutputEnabled,
		model.ReasoningEffortEnabled,
		model.ReasoningEffort,
		model.IncludeReasoning,
		model.ReasoningBudget,
		model.SupportsCacheControl,
		model.SingleMessageNoSystemPrompt,
		model.TokenEstimatePaddingPct,
		model.Providers,
	).Scan(&model.Id, &model.CreatedAt, &model.UpdatedAt)
}

func UpdateCustomModel(model *CustomModel) error {
	query := `UPDATE custom_models SET
		model_id                      = $3,
		publisher                     = $4,
		description                   = $5,
		max_tokens                    = $6,
		default_max_convo_tokens      = $7,
		max_output_tokens             = $8,
		reserved_output_tokens        = $9,
		has_image_support             = $10,
		preferred_output_format       = $11,
		system_prompt_disabled        = $12,
		role_params_disabled          = $13,
		stop_disabled                 = $14,
		predicted_output_enabled      = $15,
		reasoning_effort_enabled      = $16,
		reasoning_effort              = $17,
		include_reasoning             = $18,
		reasoning_budget              = $19,
		supports_cache_control        = $20,
		single_message_no_system_prompt = $21,
		token_estimate_padding_pct    = $22,
		providers                     = $23
	WHERE id = $1 AND org_id = $2`

	_, err := Conn.Exec(
		query,
		model.Id,
		model.OrgId,
		model.ModelId,
		model.Publisher,
		model.Description,
		model.MaxTokens,
		model.DefaultMaxConvoTokens,
		model.MaxOutputTokens,
		model.ReservedOutputTokens,
		model.HasImageSupport,
		model.PreferredOutputFormat,
		model.SystemPromptDisabled,
		model.RoleParamsDisabled,
		model.StopDisabled,
		model.PredictedOutputEnabled,
		model.ReasoningEffortEnabled,
		model.ReasoningEffort,
		model.IncludeReasoning,
		model.ReasoningBudget,
		model.SupportsCacheControl,
		model.SingleMessageNoSystemPrompt,
		model.TokenEstimatePaddingPct,
		model.Providers,
	)
	return err
}

func ListCustomModels(orgId string) ([]*CustomModel, error) {
	var models []*CustomModel
	err := Conn.Select(&models, `SELECT * FROM custom_models WHERE org_id = $1 ORDER BY created_at`, orgId)
	return models, err
}

func GetCustomModel(orgId, id string) (*CustomModel, error) {
	var model CustomModel
	err := Conn.Get(&model, `SELECT * FROM custom_models WHERE org_id = $1 AND id = $2`, orgId, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &model, nil
}

func DeleteCustomModel(orgId, id string) error {
	_, err := Conn.Exec(`DELETE FROM custom_models WHERE org_id = $1 AND id = $2`, orgId, id)
	return err
}

func CreateCustomProvider(p *CustomProvider) error {
	query := `INSERT INTO custom_providers (
		org_id,
		name,
		base_url,
		skip_auth,
		api_key_env_var,
		extra_auth_vars
	) VALUES ($1,$2,$3,$4,$5,$6)
	RETURNING id, created_at, updated_at`
	return Conn.QueryRow(
		query,
		p.OrgId,
		p.Name,
		p.BaseUrl,
		p.SkipAuth,
		p.ApiKeyEnvVar,
		p.ExtraAuthVars,
	).Scan(&p.Id, &p.CreatedAt, &p.UpdatedAt)
}

func UpdateCustomProvider(p *CustomProvider) error {
	query := `UPDATE custom_providers SET
		name            = $3,
		base_url        = $4,
		skip_auth       = $5,
		api_key_env_var = $6,
		extra_auth_vars = $7
	WHERE id = $1 AND org_id = $2`
	_, err := Conn.Exec(
		query,
		p.Id,
		p.OrgId,
		p.Name,
		p.BaseUrl,
		p.SkipAuth,
		p.ApiKeyEnvVar,
		p.ExtraAuthVars,
	)
	return err
}

func GetCustomProvider(orgId, id string) (*CustomProvider, error) {
	var provider CustomProvider
	err := Conn.Get(&provider, `SELECT * FROM custom_providers WHERE org_id = $1 AND id = $2`, orgId, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &provider, nil
}

func ListCustomProviders(orgId string) ([]*CustomProvider, error) {
	var providers []*CustomProvider
	err := Conn.Select(&providers, `SELECT * FROM custom_providers WHERE org_id = $1 ORDER BY name`, orgId)
	return providers, err
}

func DeleteCustomProvider(orgId, id string) error {
	_, err := Conn.Exec(`DELETE FROM custom_providers WHERE org_id = $1 AND id = $2`, orgId, id)
	return err
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
