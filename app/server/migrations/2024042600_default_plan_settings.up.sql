CREATE TABLE IF NOT EXISTS default_plan_settings (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  org_id UUID NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,

  plan_settings JSON,

  updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
  created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE TRIGGER update_default_plan_settings_modtime BEFORE UPDATE ON default_plan_settings FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE UNIQUE INDEX default_plan_settings_org_idx ON default_plan_settings(org_id);
