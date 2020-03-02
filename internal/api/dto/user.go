// nolint
package dto

import (
	"net/http"

	moira2 "github.com/moira-alert/moira/internal/moira"
)

type UserSettings struct {
	User
	Contacts      []moira2.ContactData      `json:"contacts"`
	Subscriptions []moira2.SubscriptionData `json:"subscriptions"`
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
