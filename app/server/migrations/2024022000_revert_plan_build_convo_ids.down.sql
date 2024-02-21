ALTER TABLE plan_builds
  RENAME COLUMN convo_message_id TO convo_message_ids;

ALTER TABLE plan_builds
  ALTER COLUMN convo_message_ids TYPE UUID[] USING ARRAY[convo_message_ids];
