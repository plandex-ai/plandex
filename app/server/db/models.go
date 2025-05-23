package db

import (
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

func UpsertCustomModel(tx *sqlx.Tx, model *CustomModel) error {
	if tx == nil {
		return fmt.Errorf("tx is nil")
	}
	query := `
INSERT INTO custom_models (
    org_id, model_id,
    publisher, description,
    max_tokens, default_max_convo_tokens, max_output_tokens, reserved_output_tokens,
    has_image_support, preferred_output_format,
    system_prompt_disabled, role_params_disabled, stop_disabled,
    predicted_output_enabled, reasoning_effort_enabled, reasoning_effort,
    include_reasoning, reasoning_budget, supports_cache_control,
    single_message_no_system_prompt, token_estimate_padding_pct,
    providers
)
VALUES (
    $1,$2,
    $3,$4,
    $5,$6,$7,$8,
    $9,$10,
    $11,$12,$13,
    $14,$15,$16,
    $17,$18,$19,
    $20,$21,
    $22
)
ON CONFLICT (org_id, model_id)
DO UPDATE SET
    publisher                     = EXCLUDED.publisher,
    description                   = EXCLUDED.description,
    max_tokens                    = EXCLUDED.max_tokens,
    default_max_convo_tokens      = EXCLUDED.default_max_convo_tokens,
    max_output_tokens             = EXCLUDED.max_output_tokens,
    reserved_output_tokens        = EXCLUDED.reserved_output_tokens,
    has_image_support             = EXCLUDED.has_image_support,
    preferred_output_format       = EXCLUDED.preferred_output_format,
    system_prompt_disabled        = EXCLUDED.system_prompt_disabled,
    role_params_disabled          = EXCLUDED.role_params_disabled,
    stop_disabled                 = EXCLUDED.stop_disabled,
    predicted_output_enabled      = EXCLUDED.predicted_output_enabled,
    reasoning_effort_enabled      = EXCLUDED.reasoning_effort_enabled,
    reasoning_effort              = EXCLUDED.reasoning_effort,
    include_reasoning             = EXCLUDED.include_reasoning,
    reasoning_budget              = EXCLUDED.reasoning_budget,
    supports_cache_control        = EXCLUDED.supports_cache_control,
    single_message_no_system_prompt = EXCLUDED.single_message_no_system_prompt,
    token_estimate_padding_pct    = EXCLUDED.token_estimate_padding_pct,
    providers                     = EXCLUDED.providers
RETURNING id, created_at, updated_at;
`

	return tx.QueryRow(
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

func ListCustomModels(orgId string) ([]*CustomModel, error) {
	var models []*CustomModel
	err := Conn.Select(&models, `SELECT * FROM custom_models WHERE org_id = $1 ORDER BY created_at`, orgId)
	return models, err
}

func ListCustomModelsForModelIds(orgId string, modelIds []string) ([]*CustomModel, error) {
	var models []*CustomModel
	query := `SELECT * FROM custom_models WHERE org_id = $1 AND model_id = ANY($2) ORDER BY created_at`
	err := Conn.Select(&models, query, orgId, pq.Array(modelIds))
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

func UpsertCustomProvider(tx *sqlx.Tx, p *CustomProvider) error {
	if tx == nil {
		return fmt.Errorf("tx is nil")
	}
	const q = `
INSERT INTO custom_providers (
	  org_id, name, base_url,
	  skip_auth, api_key_env_var, extra_auth_vars
)
VALUES (
	  $1,$2,$3,
	  $4,$5,$6
)
ON CONFLICT (org_id, name)
DO UPDATE SET
	  base_url        = EXCLUDED.base_url,
	  skip_auth       = EXCLUDED.skip_auth,
	  api_key_env_var = EXCLUDED.api_key_env_var,
	  extra_auth_vars = EXCLUDED.extra_auth_vars
RETURNING id, created_at, updated_at;
`
	return tx.QueryRow(
		q,
		p.OrgId,
		p.Name,
		p.BaseUrl,
		p.SkipAuth,
		p.ApiKeyEnvVar,
		p.ExtraAuthVars,
	).Scan(&p.Id, &p.CreatedAt, &p.UpdatedAt)
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

func ListCustomProvidersForNames(orgId string, names []string) ([]*CustomProvider, error) {
	var providers []*CustomProvider
	query := `SELECT * FROM custom_providers WHERE org_id = $1 AND name = ANY($2) ORDER BY name`
	err := Conn.Select(&providers, query, orgId, pq.Array(names))
	return providers, err
}

func DeleteCustomProvider(orgId, id string) error {
	_, err := Conn.Exec(`DELETE FROM custom_providers WHERE org_id = $1 AND id = $2`, orgId, id)
	return err
}

func UpsertModelPack(tx *sqlx.Tx, mp *ModelPack) error {
	if tx == nil {
		return fmt.Errorf("tx is nil")
	}
	const q = `
INSERT INTO model_sets (
	  org_id, name, description,
	  planner, coder, plan_summary,
	  builder, whole_file_builder, namer,
	  commit_msg, exec_status, context_loader
)
VALUES (
	  $1,$2,$3,
	  $4,$5,$6,
	  $7,$8,$9,
	  $10,$11,$12
)
ON CONFLICT (org_id, name)
DO UPDATE SET
	  description        = EXCLUDED.description,
	  planner            = EXCLUDED.planner,
	  coder              = EXCLUDED.coder,
	  plan_summary       = EXCLUDED.plan_summary,
	  builder            = EXCLUDED.builder,
	  whole_file_builder = EXCLUDED.whole_file_builder,
	  namer              = EXCLUDED.namer,
	  commit_msg         = EXCLUDED.commit_msg,
	  exec_status        = EXCLUDED.exec_status,
	  context_loader     = EXCLUDED.context_loader
RETURNING id, created_at;
`
	return tx.QueryRow(
		q,
		mp.OrgId,
		mp.Name,
		mp.Description,
		mp.Planner,
		mp.Coder,
		mp.PlanSummary,
		mp.Builder,
		mp.WholeFileBuilder,
		mp.Namer,
		mp.CommitMsg,
		mp.ExecStatus,
		mp.Architect,
	).Scan(&mp.Id, &mp.CreatedAt)
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

func ListModelPacksForNames(orgId string, names []string) ([]*ModelPack, error) {
	var modelPacks []*ModelPack
	query := `SELECT * FROM model_sets WHERE org_id = $1 AND name = ANY($2)`
	err := Conn.Select(&modelPacks, query, orgId, names)
	return modelPacks, err
}

func DeleteModelPack(setId string) error {
	query := `DELETE FROM model_sets WHERE id = $1`
	_, err := Conn.Exec(query, setId)

	if err != nil {
		return fmt.Errorf("error deleting model pack: %v", err)
	}

	return nil
}
