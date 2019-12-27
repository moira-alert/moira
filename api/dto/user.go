// nolint
package dto

import (
	"net/http"

	"github.com/moira-alert/moira"
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
	Login string `json:"login"`
}

func (*User) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
