ALTER TABLE custom_models 
  ADD COLUMN max_output_tokens INTEGER NOT NULL,
  ADD COLUMN reserved_output_tokens INTEGER NOT NULL,
  ADD COLUMN model_id VARCHAR(255) NOT NULL;