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
	Type  string `json:"type"`
	Value string `json:"value"`
	ID    string `json:"id,omitempty"`
	User  string `json:"user,omitempty"`
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
