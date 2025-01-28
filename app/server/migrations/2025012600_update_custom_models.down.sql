ALTER TABLE custom_models DROP COLUMN preferred_output_format;

ALTER TABLE custom_models ADD COLUMN has_streaming BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE custom_models ADD COLUMN has_function_calling BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE custom_models ADD COLUMN has_streaming_function_calls BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE custom_models ADD COLUMN has_json_mode BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE custom_models ADD COLUMN is_openai_compatible BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE custom_models DROP COLUMN has_image_support;
