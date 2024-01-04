package shared

type AuthHeader struct {
	Token string `json:"token"`
	OrgId string `json:"orgId"`
}
