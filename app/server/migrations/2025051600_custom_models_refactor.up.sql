
BEGIN;
ALTER TABLE custom_models RENAME TO custom_models_legacy;

CREATE TABLE IF NOT EXISTS custom_models (
  id                       UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  org_id                   UUID NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,

  model_id                 VARCHAR(255) NOT NULL,
  description              TEXT,
  publisher                VARCHAR(64) NOT NULL DEFAULT '',

  max_tokens               INTEGER NOT NULL,
  default_max_convo_tokens INTEGER NOT NULL,
  max_output_tokens        INTEGER NOT NULL,
  reserved_output_tokens   INTEGER NOT NULL,

  has_image_support        BOOLEAN NOT NULL DEFAULT FALSE,
  preferred_output_format  VARCHAR(32) NOT NULL DEFAULT 'xml',

  system_prompt_disabled       BOOLEAN NOT NULL DEFAULT FALSE,
  role_params_disabled         BOOLEAN NOT NULL DEFAULT FALSE,
  stop_disabled                BOOLEAN NOT NULL DEFAULT FALSE,
  predicted_output_enabled     BOOLEAN NOT NULL DEFAULT FALSE,
  reasoning_effort_enabled     BOOLEAN NOT NULL DEFAULT FALSE,
  reasoning_effort             VARCHAR(32) NOT NULL DEFAULT '',
  include_reasoning            BOOLEAN NOT NULL DEFAULT FALSE,
  reasoning_budget             INTEGER NOT NULL DEFAULT 0,
  supports_cache_control       BOOLEAN NOT NULL DEFAULT FALSE,
  single_message_no_system_prompt BOOLEAN NOT NULL DEFAULT FALSE,
  token_estimate_padding_pct   FLOAT NOT NULL DEFAULT 0.0,

  providers JSON NOT NULL DEFAULT '[]',

  created_at               TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at               TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS cmv_org_idx ON custom_models(org_id);
CREATE UNIQUE INDEX IF NOT EXISTS cmv_unique_idx ON custom_models(org_id, model_id);

CREATE TRIGGER cmv_modtime BEFORE UPDATE ON custom_models
  FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TABLE IF NOT EXISTS custom_providers (
  id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  org_id          UUID NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,

  name            VARCHAR(255) NOT NULL,          
  base_url        VARCHAR(255) NOT NULL,
  skip_auth       BOOLEAN NOT NULL DEFAULT FALSE,
  api_key_env_var VARCHAR(255),

  extra_auth_vars JSON NOT NULL DEFAULT '[]',

  created_at      TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMP NOT NULL DEFAULT NOW()                          
);
CREATE INDEX IF NOT EXISTS cp_org_idx ON custom_providers(org_id);
CREATE UNIQUE INDEX IF NOT EXISTS cp_unique_idx ON custom_providers(org_id, name);
CREATE TRIGGER cp_modtime BEFORE UPDATE ON custom_providers
  FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

/* ---- migrate base rows into the new custom_models ------------------ */
INSERT INTO custom_models (
    id, org_id, model_id, description,
    max_tokens, default_max_convo_tokens,
    max_output_tokens, reserved_output_tokens,
    has_image_support, preferred_output_format,

    providers,                 -- <-- aggregated JSON array
    created_at, updated_at
)
SELECT
    id, org_id, model_id, description,
    max_tokens, default_max_convo_tokens,
    max_output_tokens, reserved_output_tokens,
    has_image_support, preferred_output_format,

    /* -------- build a one-element providers array -------- */
    json_build_array(
        json_build_object(
            'provider',        provider,
            'custom_provider', custom_provider,
            'model_name',      model_name
        )
    )::json,

    created_at, updated_at
FROM custom_models_legacy
ON CONFLICT (org_id, model_id) DO NOTHING;

/* ---- migrate unique custom providers ------------------------------- */
WITH src AS (
  SELECT DISTINCT
         org_id,
         custom_provider AS name,
         base_url,
         api_key_env_var
  FROM   custom_models_legacy
  WHERE  custom_provider IS NOT NULL
)
INSERT INTO custom_providers (org_id, name, base_url, api_key_env_var)
SELECT org_id, name, base_url, api_key_env_var
FROM   src
ON CONFLICT (org_id, name) DO NOTHING;

COMMIT;