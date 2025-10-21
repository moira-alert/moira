// nolint
package dto

import (
	"fmt"
	"net/http"

	"github.com/moira-alert/moira"
)

type ContactList struct {
	List []TeamContact `json:"list" binding:"required"`
}

func (*ContactList) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

type Contact struct {
	Type         string `json:"type" binding:"required" example:"mail"`
	Name         string `json:"name,omitempty" example:"Mail Alerts"`
	Value        string `json:"value" binding:"required" example:"devops@example.com"`
	ID           string `json:"id" binding:"required" example:"1dd38765-c5be-418d-81fa-7a5f879c2315"`
	User         string `json:"user,omitempty" example:""`
	TeamID       string `json:"team_id,omitempty"`
	ExtraMessage string `json:"extra_message,omitempty"`
}

// NewContact init Contact with data from moira.ContactData.
func NewContact(data moira.ContactData) Contact {
	return Contact{
		Type:         data.Type,
		Name:         data.Name,
		Value:        data.Value,
		ID:           data.ID,
		User:         data.User,
		TeamID:       data.Team,
		ExtraMessage: data.ExtraMessage,
	}
}

func (*Contact) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

const maxExtraMessageLen = 100

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
	if len(contact.ExtraMessage) > maxExtraMessageLen {
		return fmt.Errorf("contact extra message must not be longer then %d characters long", maxExtraMessageLen)
	}
	return nil
}

// ContactNoisiness represents Contact with amount of events for this contact.
type ContactNoisiness struct {
	Contact
	// EventsCount for the contact.
	EventsCount uint64 `json:"events_count" binding:"required"`
}

func (*ContactNoisiness) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

// ContactNoisinessList represents list of ContactNoisiness.
type ContactNoisinessList ListDTO[*ContactNoisiness]

func (*ContactNoisinessList) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

// ContactScore represents the score details of a contact.
type ContactScore struct {
	// ScorePercent is the percentage score of successful transactions.
	ScorePercent *uint8 `json:"score_percent,omitempty"`
	// LastErrMessage is the last error message encountered.
	LastErrMessage string `json:"last_err,omitempty"`
	// LastErrTimestamp is the timestamp of the last error.
	LastErrTimestamp uint64 `json:"last_err_timestamp,omitempty"`
	// Status is the current status of the contact.
	Status string `json:"status,omitempty"`
}

func NewContactScore(data *moira.ContactScore) *ContactScore {
	if data == nil {
		return nil
	}

	return &ContactScore{
		Status: string(data.Status),
		LastErrMessage: data.LastErrorMsg,
		LastErrTimestamp: data.LastErrorTimestamp,
		ScorePercent: moira.CalculatePercentage(data.SuccessTXCount, data.AllTXCount),
	}
}

// ContactWithScore represents a contact with an associated score.
type ContactWithScore struct {
	Contact
	Score *ContactScore `json:"score,omitempty" extensions:"x-nullable"`
}
