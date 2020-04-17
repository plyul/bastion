package auth

// By default, ADFS return only 'upn' and 'sid' claims
// Those additional claims must be configured via 'Issuance Transform Rules'
type UserClaims struct {
	SID         string `json:"sid"`
	UPN         string `json:"upn"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
}
