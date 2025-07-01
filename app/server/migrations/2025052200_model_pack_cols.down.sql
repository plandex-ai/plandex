ALTER TABLE model_sets
  DROP COLUMN IF EXISTS updated_at;

DROP TRIGGER IF EXISTS model_set_modtime ON model_sets;

DROP INDEX IF EXISTS model_set_unique_idx;
