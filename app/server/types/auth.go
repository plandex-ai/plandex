package types

import "plandex-server/db"

type ServerAuth struct {
	AuthToken   *db.AuthToken
	User        *db.User
	OrgId       string
	Permissions map[Permission]bool
}

func (a *ServerAuth) HasPermission(permission Permission) bool {
	if a.Permissions == nil {
		return false
	}

	return a.Permissions[permission]
}

type Permission string

const (
	PermissionDeleteOrg             Permission = "delete_org"
	PermissionManageEmailDomainAuth Permission = "manage_email_domain_auth"
	PermissionManageBilling         Permission = "manage_billing"
	PermissionInviteUser            Permission = "invite_user"
	PermissionRemoveUser            Permission = "remove_user"
	PermissionSetUserRole           Permission = "set_user_role"
	PermissionListOrgRoles          Permission = "list_org_roles"
	PermissionCreateProject         Permission = "create_project"
	PermissionRenameAnyProject      Permission = "rename_any_project"
	PermissionDeleteAnyProject      Permission = "delete_any_project"
	PermissionCreatePlan            Permission = "create_plan"
	PermissionManageAnyPlanShares   Permission = "manage_any_plan_shares"
	PermissionRenameAnyPlan         Permission = "rename_any_plan"
	PermissionDeleteAnyPlan         Permission = "delete_any_plan"
	PermissionUpdateAnyPlan         Permission = "update_any_plan"
	PermissionArchiveAnyPlan        Permission = "archive_any_plan"
)
