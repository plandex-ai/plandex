CREATE TABLE IF NOT EXISTS model_streams (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  org_id UUID NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
  plan_id UUID NOT NULL REFERENCES plans(id) ON DELETE CASCADE,
  branch VARCHAR(255) NOT NULL,
  internal_ip VARCHAR(45) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  finished_at TIMESTAMP
);

CREATE UNIQUE INDEX model_streams_plan_idx ON model_streams(plan_id, branch, finished_at);

-- CREATE TABLE IF NOT EXISTS model_stream_subscriptions (
--   id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
--   org_id UUID NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
--   plan_id UUID NOT NULL REFERENCES plans(id) ON DELETE CASCADE,
--   model_stream_id UUID NOT NULL REFERENCES model_streams(id) ON DELETE CASCADE,
--   user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
--   user_ip VARCHAR(45) NOT NULL,
--   created_at TIMESTAMP NOT NULL DEFAULT NOW()
--   finished_at TIMESTAMP
-- );

