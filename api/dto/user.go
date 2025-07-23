// nolint
package dto

import (
	"net/http"
	"github.com/moira-alert/moira/api"
)

type UserSettings struct {
	User
	Contacts      []ContactWithScore      `json:"contacts"`
	Subscriptions []Subscription `json:"subscriptions"`
}

func (*UserSettings) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

type User struct {
	Login       string   `json:"login" example:"john"`
	Role        api.Role `json:"role,omitempty" example:"user"`
	AuthEnabled bool     `json:"auth_enabled,omitempty" example:"true"`
}

func (*User) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
