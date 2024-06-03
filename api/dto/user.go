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

type User struct {
	Login       string   `json:"login" example:"john"`
	Role        api.Role `json:"role,omitempty" example:"user"`
	AuthEnabled bool     `json:"auth_enabled,omitempty" example:"true"`
}

func (*User) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
