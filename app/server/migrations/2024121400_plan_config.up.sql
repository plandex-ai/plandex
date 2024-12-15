ALTER TABLE plans ADD COLUMN IF NOT EXISTS plan_config JSON;

CREATE TABLE IF NOT EXISTS default_plan_config (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

  plan_config JSON,

  updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
  created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE TRIGGER update_default_plan_config_modtime BEFORE UPDATE ON default_plan_config FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE UNIQUE INDEX default_plan_config_user_idx ON default_plan_config(user_id);
