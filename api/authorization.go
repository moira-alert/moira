package api

// Authorization contains authorization configuration.
type Authorization struct {
	AdminList           map[string]struct{}
	Enabled             bool
	AllowedContactTypes map[string]struct{}
}

// IsEnabled returns true if auth is enabled and false otherwise.
func (auth *Authorization) IsEnabled() bool {
	return auth.Enabled
}

// IsAdmin checks whether given user is considered an administrator.
func (auth *Authorization) IsAdmin(login string) bool {
	if !auth.IsEnabled() {
		return false
	}
	_, ok := auth.AdminList[login]
	return ok
}

// The Role is an enumeration that represents the scope of user's permissions.
type Role string

var (
	RoleUndefined Role = ""
	RoleUser      Role = "user"
	RoleAdmin     Role = "admin"
)

// Returns the role of the given user.
func (auth *Authorization) GetRole(login string) Role {
	if !auth.IsEnabled() {
		return RoleUndefined
	}
	if auth.IsAdmin(login) {
		return RoleAdmin
	}
	return RoleUser
}
