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
	Score ContactScore `json:"score,omitempty"`
}

// NewContact init Contact with data from moira.ContactData.
func NewContact(data moira.ContactData, score moira.ContactScore) Contact {
	return Contact{
		Type:   data.Type,
		Name:   data.Name,
		Value:  data.Value,
		ID:     data.ID,
		User:   data.User,
		TeamID: data.Team,
		Score: ContactScore{
			ScorePercent: moira.CalculatePercentage(score.SuccessTXCount, score.AllTXCount),
			LastErrMessage: score.LastErrorMsg,
			LastErrTimestamp: score.LastErrorTimestamp,
			Status: string(score.Status),
		},
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

type ContactScore struct {
	ScorePercent *uint8 `json:"score_percent,omitempty"`
	LastErrMessage string `json:"last_err,omitempty"`
	LastErrTimestamp uint64 `json:"last_err_timestamp,omitempty"`
	Status string `json:"status,omitempty"`
}

