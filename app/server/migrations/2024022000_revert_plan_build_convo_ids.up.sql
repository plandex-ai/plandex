ALTER TABLE plan_builds
  RENAME COLUMN convo_message_ids TO convo_message_id;

ALTER TABLE plan_builds
  ALTER COLUMN convo_message_id TYPE UUID USING (convo_message_id[1]);