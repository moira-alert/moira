// nolint
package dto

import (
	"net/http"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
)

type UserSettings struct {
	User
	Contacts      []moira.ContactData      `json:"contacts"`
	Subscriptions []moira.SubscriptionData `json:"subscriptions"`
}

func (*UserSettings) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

type Role string

var (
	RoleUndefined Role = ""
	RoleUser      Role = "user"
	RoleAdmin     Role = "admin"
)

func GetRole(login string, auth *api.Authorization) Role {
	if !auth.IsEnabled() {
		return RoleUndefined
	}
	if auth.IsAdmin(login) {
		return RoleAdmin
	}
	return RoleUser
}

type User struct {
	Login       string `json:"login" example:"john"`
	Role        Role   `json:"role,omitempty" example:"user"`
	AuthEnabled bool   `json:"auth_enabled" example:"true"`
}

func (*User) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
