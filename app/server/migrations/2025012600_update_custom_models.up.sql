ALTER TABLE custom_models ADD COLUMN preferred_output_format VARCHAR(32) NOT NULL DEFAULT 'xml';

ALTER TABLE custom_models DROP COLUMN has_streaming_function_calls;
ALTER TABLE custom_models DROP COLUMN has_json_mode;
ALTER TABLE custom_models DROP COLUMN has_streaming;
ALTER TABLE custom_models DROP COLUMN has_function_calling;
ALTER TABLE custom_models DROP COLUMN is_openai_compatible;

ALTER TABLE custom_models ADD COLUMN has_image_support BOOLEAN NOT NULL DEFAULT FALSE;