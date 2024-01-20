
CREATE TABLE IF NOT EXISTS org_roles (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  org_id UUID REFERENCES orgs(id) ON DELETE CASCADE,
  name VARCHAR(255) NOT NULL,
  label VARCHAR(255) NOT NULL,
  description TEXT NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE TRIGGER update_org_roles_modtime BEFORE UPDATE ON org_roles FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE UNIQUE INDEX org_roles_org_idx ON org_roles(org_id, name);

ALTER TABLE orgs_users ADD COLUMN org_role_id UUID NOT NULL REFERENCES org_roles(id) ON DELETE RESTRICT;
CREATE INDEX orgs_users_org_role_idx ON orgs_users(org_id, org_role_id);

ALTER TABLE invites ADD COLUMN org_role_id UUID NOT NULL REFERENCES org_roles(id) ON DELETE RESTRICT;
CREATE INDEX invites_org_role_idx ON invites(org_id, org_role_id);

CREATE TABLE IF NOT EXISTS permissions (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  name VARCHAR(255) NOT NULL,
  description TEXT NOT NULL,
  resource_id UUID,
  created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS org_roles_permissions (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  org_role_id UUID NOT NULL REFERENCES org_roles(id) ON DELETE CASCADE,
  permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
  created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

INSERT INTO org_roles (name, label, description) VALUES
  ('owner', 'Owner', 'Can read and update any plan, invite other owners/admins/members, manage email domain auth, manage billing, read audit logs, delete the org'),
  ('billing_admin', 'Billing Admin', 'Can manage billing'),
  ('admin', 'Admin', 'Can read and update any plan, invite other admins/members'),
  ('member', 'Member', 'Can read and update their own plans or plans shared with them');

DO $$
DECLARE
  owner_org_role_id UUID;
  billing_admin_org_role_id UUID;
  admin_org_role_id UUID;
  member_org_role_id UUID;
BEGIN
  SELECT id INTO owner_org_role_id FROM org_roles WHERE org_id IS NULL AND name = 'owner';
  SELECT id INTO billing_admin_org_role_id FROM org_roles WHERE org_id IS NULL AND name = 'billing_admin';
  SELECT id INTO admin_org_role_id FROM org_roles WHERE org_id IS NULL AND name = 'admin';
  SELECT id INTO member_org_role_id FROM org_roles WHERE org_id IS NULL AND name = 'member';

  INSERT INTO permissions (name, description, resource_id) VALUES
    ('delete_org', 'Delete an org', NULL),
    ('manage_email_domain_auth', 'Configure whether orgs_users from the org''s email domain are auto-admitted to org', NULL),
    ('manage_billing', 'Manage an org''s billing', NULL),
    
    ('invite_user', 'Invite owners to an org', owner_org_role_id),
    ('invite_user', 'Invite admins to an org', admin_org_role_id),
    ('invite_user', 'Invite billing admins to an org', billing_admin_org_role_id),
    ('invite_user', 'Invite members to an org', member_org_role_id),
    
    ('remove_user', 'Remove owners from an org', owner_org_role_id),
    ('remove_user', 'Remove admins from an org', admin_org_role_id),
    ('remove_user', 'Remove billing admins from an org', billing_admin_org_role_id),
    ('remove_user', 'Remove members from an org', member_org_role_id),
    
    ('set_user_role', 'Update an owner''s role in an org', owner_org_role_id),
    ('set_user_role', 'Update an admin''s role in an org', admin_org_role_id),
    ('set_user_role', 'Update a billing admin''s role in an org', billing_admin_org_role_id),
    ('set_user_role', 'Update a member''s role in an org', member_org_role_id),

    ('list_org_roles', 'List org roles', NULL),
    
    ('create_project', 'Create a project', NULL),
    ('rename_any_project', 'Rename a project', NULL),
    ('delete_any_project', 'Delete a project', NULL),

    ('create_plan', 'Create a plan', NULL),

    ('manage_any_plan_shares', 'Unshare a plan any user shared', NULL),
    ('rename_any_plan', 'Rename a plan', NULL),
    ('delete_any_plan', 'Delete a plan', NULL),
    ('update_any_plan', 'Update a plan', NULL),
    ('archive_any_plan', 'Archive a plan', NULL);
END $$;

-- Insert all permissions for the 'org owner' role
INSERT INTO org_roles_permissions (org_role_id, permission_id)
SELECT 
    (SELECT id FROM org_roles WHERE name = 'owner') AS org_role_id, 
    p.id AS permission_id
FROM 
    permissions p;

-- Insert all permissions except specific ones and those exclusive to 'owner' or 'billing admin' for the 'org admin' role
INSERT INTO org_roles_permissions (org_role_id, permission_id)
SELECT 
    (SELECT id FROM org_roles WHERE name = 'admin') AS org_role_id, 
    p.id AS permission_id
FROM 
    permissions p
WHERE 
    p.name NOT IN ('delete_org', 'manage_email_domain_auth', 'manage_billing')
    AND NOT EXISTS (
        SELECT 1 FROM permissions p2
        WHERE p2.resource_id IN (SELECT id FROM org_roles WHERE name IN ('owner', 'billing_admin'))
        AND p2.id = p.id
    );

INSERT INTO org_roles_permissions (org_role_id, permission_id)
SELECT 
    (SELECT id FROM org_roles WHERE name = 'billing_admin') AS org_role_id, 
    p.id AS permission_id
FROM
    permissions p
WHERE 
    p.name IN (
      'manage_billing'
    );

INSERT INTO org_roles_permissions (org_role_id, permission_id)
SELECT 
    (SELECT id FROM org_roles WHERE name = 'member') AS org_role_id, 
    p.id AS permission_id
FROM
    permissions p
WHERE 
    p.name IN (
      'create_project',
      'create_plan'
    );
