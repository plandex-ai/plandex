
CREATE UNIQUE INDEX repo_locks_single_write_lock
  ON repo_locks(plan_id)
  WHERE (scope = 'w');

CREATE TABLE IF NOT EXISTS lockable_plan_ids (
  plan_id UUID NOT NULL PRIMARY KEY REFERENCES plans(id) ON DELETE CASCADE
);