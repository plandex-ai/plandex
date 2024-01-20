package types

import "plandex-server/db"

type ServerAuth struct {
	AuthToken   *db.AuthToken
	User        *db.User
	OrgId       string
	Permissions map[string]bool
}

func (a *ServerAuth) HasPermission(permission string) bool {
	if a.Permissions == nil {
		return false
	}

	return a.Permissions[permission]
}
