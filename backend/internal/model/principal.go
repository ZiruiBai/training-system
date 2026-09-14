package model

// Principal is the authenticated caller identity derived from the session.
type Principal struct {
	UserID      uint
	Username    string
	Role        Role
	EmployeeID  *uint
	DisplayName string
}

// IsRole returns whether the principal holds one of the roles.
func (p *Principal) IsRole(roles ...Role) bool {
	if p == nil {
		return false
	}
	for _, r := range roles {
		if p.Role == r {
			return true
		}
	}
	return false
}
