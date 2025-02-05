// nolint
package dto

import (
	"fmt"
	"net/http"

	"github.com/moira-alert/moira"
)

type ContactList struct {
	List []TeamContact `json:"list"`
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

// FromMoiraContactData converts moira.ContactData data into Contact.
func FromMoiraContactData(data moira.ContactData) Contact {
	return Contact{
		Type:   data.Type,
		Name:   data.Name,
		Value:  data.Value,
		ID:     data.ID,
		User:   data.User,
		TeamID: data.Team,
	}
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
	if contact.User != "" && contact.TeamID != "" {
		return fmt.Errorf("contact cannot have both the user field and the team_id field filled in")
	}
	return nil
}

// ContactNoisiness represents Contact with amount of events for this contact.
type ContactNoisiness struct {
	Contact
	// EventsCount for the contact.
	EventsCount uint64 `json:"events_count"`
}

func (*ContactNoisiness) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

// ContactNoisinessList represents list of ContactNoisiness.
type ContactNoisinessList ListDTO[*ContactNoisiness]

func (*ContactNoisinessList) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
