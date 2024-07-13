CREATE TABLE IF NOT EXISTS users_projects (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  org_id UUID NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  last_active_plan_id UUID REFERENCES plans(id),
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE TRIGGER update_users_projects_modtime BEFORE UPDATE ON users_projects FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE INDEX users_projects_idx ON users_projects(user_id, org_id, project_id);