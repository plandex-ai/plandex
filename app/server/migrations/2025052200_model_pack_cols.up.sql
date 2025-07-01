ALTER TABLE model_sets
  ADD COLUMN updated_at TIMESTAMP NOT NULL DEFAULT NOW();

CREATE TRIGGER model_set_modtime BEFORE UPDATE ON model_sets
  FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE UNIQUE INDEX IF NOT EXISTS model_set_unique_idx ON model_sets(org_id, name);