package db

import (
	"context"
	"database/sql"
	"github.com/plandex/plandex/shared"
	"log"
)

// CreateCustomModel inserts a new custom model into the database.
func CreateCustomModel(ctx context.Context, db *sql.DB, model shared.CustomModel) error {
	query := `INSERT INTO custom_models (org_id, provider, custom_provider, base_url, model_name, description, max_tokens, api_key_env_var, is_openai_compatible, has_json_mode, has_streaming, has_function_calling, has_streaming_function_calls, default_max_convo_tokens, default_reserved_output_tokens) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`
	_, err := db.ExecContext(ctx, query, model.OrgId, model.Provider, model.CustomProvider, model.BaseUrl, model.ModelName, model.Description, model.MaxTokens, model.ApiKeyEnvVar, model.IsOpenAICompatible, model.HasJsonResponseMode, model.HasStreaming, model.HasFunctionCalling, model.HasStreamingFunctionCalls, model.DefaultMaxConvoTokens, model.DefaultReservedOutputTokens)
	return err
}

// ListCustomModels retrieves all custom models from the database.
func ListCustomModels(ctx context.Context, db *sql.DB, orgId string) ([]shared.CustomModel, error) {
	query := `SELECT * FROM custom_models WHERE org_id = $1`
	rows, err := db.QueryContext(ctx, query, orgId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var models []shared.CustomModel
	for rows.Next() {
		var model shared.CustomModel
		if err := rows.Scan(&model.Id, &model.OrgId, &model.Provider, &model.CustomProvider, &model.BaseUrl, &model.ModelName, &model.Description, &model.MaxTokens, &model.ApiKeyEnvVar, &model.IsOpenAICompatible, &model.HasJsonResponseMode, &model.HasStreaming, &model.HasFunctionCalling, &model.HasStreamingFunctionCalls, &model.DefaultMaxConvoTokens, &model.DefaultReservedOutputTokens, &model.CreatedAt, &model.UpdatedAt); err != nil {
			return nil, err
		}
		models = append(models, model)
	}
	return models, nil
}

// DeleteCustomModel removes a custom model from the database.
func DeleteCustomModel(ctx context.Context, db *sql.DB, modelId string) error {
	query := `DELETE FROM custom_models WHERE id = $1`
	_, err := db.ExecContext(ctx, query, modelId)
	return err
}

// CreateModelSet inserts a new model set into the database.
func CreateModelSet(ctx context.Context, db *sql.DB, set shared.ModelSet) error {
	query := `INSERT INTO model_sets (org_id, name, description, planner, plan_summary, builder, namer, commit_msg, exec_status) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err := db.ExecContext(ctx, query, set.OrgId, set.Name, set.Description, set.Planner, set.PlanSummary, set.Builder, set.Namer, set.CommitMsg, set.ExecStatus)
	return err
}

// ListModelSets retrieves all model sets from the database.
func ListModelSets(ctx context.Context, db *sql.DB, orgId string) ([]shared.ModelSet, error) {
	query := `SELECT * FROM model_sets WHERE org_id = $1`
	rows, err := db.QueryContext(ctx, query, orgId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sets []shared.ModelSet
	for rows.Next() {
		var set shared.ModelSet
		if err := rows.Scan(&set.Id, &set.OrgId, &set.Name, &set.Description, &set.Planner, &set.PlanSummary, &set.Builder, &set.Namer, &set.CommitMsg, &set.ExecStatus, &set.CreatedAt); err != nil {
			return nil, err
		}
		sets = append(sets, set)
	}
	return sets, nil
}

// DeleteModelSet removes a model set from the database.
func DeleteModelSet(ctx context.Context, db *sql.DB, setId string) error {
	query := `DELETE FROM model_sets WHERE id = $1`
	_, err := db.ExecContext(ctx, query, setId)
	return err
}
