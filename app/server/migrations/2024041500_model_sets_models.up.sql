CREATE TABLE IF NOT EXISTS model_sets (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  org_id UUID NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,

  name VARCHAR(255) NOT NULL,
  description TEXT,

  planner JSON,
  plan_summary JSON,
  builder JSON,
  namer JSON,
  commit_msg JSON,
  exec_status JSON,

  created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX model_sets_org_idx ON model_sets(org_id);

CREATE TABLE IF NOT EXISTS custom_models (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  org_id UUID NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
  
  provider VARCHAR(255) NOT NULL,
  custom_provider VARCHAR(255),
  base_url VARCHAR(255) NOT NULL,
  model_name VARCHAR(255) NOT NULL,
  description TEXT,
  max_tokens INTEGER NOT NULL,
  api_key_env_var VARCHAR(255),

  is_openai_compatible BOOLEAN NOT NULL,
  has_json_mode BOOLEAN NOT NULL,
  has_streaming BOOLEAN NOT NULL,
  has_function_calling BOOLEAN NOT NULL,
  has_streaming_function_calls BOOLEAN NOT NULL,

  default_max_convo_tokens INTEGER NOT NULL,
  default_reserved_output_tokens INTEGER NOT NULL,

  updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
  created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TRIGGER update_custom_models_modtime BEFORE UPDATE ON custom_models FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE INDEX custom_models_org_idx ON custom_models(org_id);
