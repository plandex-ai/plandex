package types

import "plandex-server/db"

type ServerAuth struct {
	AuthToken *db.AuthToken
	User      *db.User
	OrgId     string
}
