-- Revert user_id to NOT NULL if no NULL values exist
DO $$ 
BEGIN
    -- Check for NULL values in user_id
    IF EXISTS (SELECT 1 FROM repo_locks WHERE user_id IS NULL) THEN
        RAISE EXCEPTION 'Cannot revert to NOT NULL, as there are rows with NULL values in user_id.';
    ELSE
        -- Proceed with setting the columns to NOT NULL
        ALTER TABLE repo_locks
          ALTER COLUMN user_id SET NOT NULL;
    END IF;
END $$;
