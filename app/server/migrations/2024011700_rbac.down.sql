ALTER TABLE orgs_users DROP COLUMN org_role_id;
ALTER TABLE invites DROP COLUMN org_role_id;

DROP TABLE IF EXISTS org_roles_permissions;
DROP TABLE IF EXISTS permissions;
DROP TABLE IF EXISTS org_roles;


