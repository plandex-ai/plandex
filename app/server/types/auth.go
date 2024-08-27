package types

import (
	"plandex-server/db"

	"github.com/plandex/plandex/shared"
)

type ServerAuth struct {
	AuthToken   *db.AuthToken
	User        *db.User
	OrgId       string
	Permissions shared.Permissions
}

func (a *ServerAuth) HasPermission(permission shared.Permission) bool {
	return a.Permissions.HasPermission(permission)
}

func (a *ServerAuth) HasPermissionForResource(permission shared.Permission, resourceId string) bool {
	return a.Permissions.HasPermissionForResource(permission, resourceId)
}
