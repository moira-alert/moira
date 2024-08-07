// nolint
package dto

import (
	"fmt"
	"net/http"

	"github.com/moira-alert/moira"
)

type ContactList struct {
	List []*moira.ContactData `json:"list"`
}

func (*ContactList) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

type Contact struct {
	Type   string `json:"type" example:"mail"`
	Name   string `json:"name,omitempty" example:"Mail Alerts"`
	Value  string `json:"value" example:"devops@example.com"`
	ID     string `json:"id,omitempty" example:"1dd38765-c5be-418d-81fa-7a5f879c2315"`
	User   string `json:"user,omitempty" example:""`
	TeamID string `json:"team_id,omitempty"`
}

func (*Contact) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

func (contact *Contact) Bind(r *http.Request) error {
	if contact.Type == "" {
		return fmt.Errorf("contact type can not be empty")
	}
	if contact.Value == "" {
		return fmt.Errorf("contact value of type %s can not be empty", contact.Type)
	}
	return nil
}
