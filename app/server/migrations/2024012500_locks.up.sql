CREATE UNLOGGED TABLE IF NOT EXISTS repo_locks (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  org_id UUID NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  plan_id UUID NOT NULL REFERENCES plans(id) ON DELETE CASCADE,
  plan_build_id UUID REFERENCES plan_builds(id) ON DELETE CASCADE,
  scope VARCHAR(1) NOT NULL,
  branch VARCHAR(255),
  created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX repo_locks_plan_idx ON repo_locks(plan_id);
