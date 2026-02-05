package dto

import (
	"errors"
	"fmt"
	"net/http"
	"unicode/utf8"

	"github.com/moira-alert/moira/api/middleware"

	"github.com/moira-alert/moira"
)

var errEmptyTeamName = errors.New("team name cannot be empty")

// TeamModel is a structure that represents team entity in HTTP transfer.
type TeamModel struct {
	ID          string `json:"id" binding:"required" example:"d5d98eb3-ee18-4f75-9364-244f67e23b54"`
	Name        string `json:"name" binding:"required" example:"Infrastructure Team"`
	Description string `json:"description" example:"Team that holds all members of infrastructure division"`
}

// NewTeamModel is a constructor function that creates a new TeamModel using moira.Team.
func NewTeamModel(team moira.Team) TeamModel {
	return TeamModel{
		ID:          team.ID,
		Name:        team.Name,
		Description: team.Description,
	}
}

// Bind is a method that implements Binder interface from chi and checks that validity of data in request.
func (t TeamModel) Bind(request *http.Request) error {
	limits := middleware.GetLimits(request)

	if t.Name == "" {
		return errEmptyTeamName
	}

	if utf8.RuneCountInString(t.Name) > limits.Team.MaxNameSize {
		return fmt.Errorf("team name cannot be longer than %d characters", limits.Team.MaxNameSize)
	}

	if utf8.RuneCountInString(t.Description) > limits.Team.MaxDescriptionSize {
		return fmt.Errorf("team description cannot be longer than %d characters", limits.Team.MaxDescriptionSize)
	}

	return nil
}

// Render is a function that implements chi Renderer interface for TeamModel.
func (TeamModel) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

// ToMoiraTeam is a method that converts dto.Team to general moira.Team datatype.
func (t TeamModel) ToMoiraTeam() moira.Team {
	return moira.Team{
		ID:          t.ID,
		Name:        t.Name,
		Description: t.Description,
	}
}

// SaveTeamResponse is a structure to return team creation result in HTTP response.
type SaveTeamResponse struct {
	ID string `json:"id" binding:"required" example:"d5d98eb3-ee18-4f75-9364-244f67e23b54"`
}

// Render is a function that implements chi Renderer interface for SaveTeamResponse.
func (SaveTeamResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

// UserTeams is a structure that represents a set of teams of user.
type UserTeams struct {
	Teams []TeamModel `json:"teams" binding:"required"`
}

// Render is a function that implements chi Renderer interface for UserTeams.
func (UserTeams) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

// TeamMembers is a structure that represents a team members in HTTP transfer.
type TeamMembers struct {
	Usernames []string `json:"usernames" binding:"required" example:"anonymous"`
}

// Bind is a method that implements Binder interface from chi and checks that validity of data in request.
func (m TeamMembers) Bind(request *http.Request) error {
	if len(m.Usernames) == 0 {
		return fmt.Errorf("at least one user should be specified")
	}

	return nil
}

// Render is a function that implements chi Renderer interface for TeamMembers.
func (TeamMembers) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

// TeamSettings is a structure that contains info about team: contacts and subscriptions.
type TeamSettings struct {
	TeamID        string                   `json:"team_id" binding:"required" example:"d5d98eb3-ee18-4f75-9364-244f67e23b54"`
	Contacts      []TeamContactWithScore   `json:"contacts" binding:"required"`
	Subscriptions []moira.SubscriptionData `json:"subscriptions" binding:"required"`
}

// Render is a function that implements chi Renderer interface for TeamSettings.
func (TeamSettings) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

// nolint TODO: Replace with dto.Contact after the next release.
type TeamContact struct {
	Type   string `json:"type" binding:"required" example:"mail"`
	Name   string `json:"name,omitempty" example:"Mail Alerts"`
	Value  string `json:"value" binding:"required" example:"devops@example.com"`
	ID     string `json:"id" binding:"required" example:"1dd38765-c5be-418d-81fa-7a5f879c2315"`
	User   string `json:"user,omitempty" example:""`
	TeamID string `json:"team_id,omitempty"`
	// This field is deprecated
	Team         string `json:"team,omitempty"`
	ExtraMessage string `json:"extra_message,omitempty"`
}

// MakeTeamContact converts moira.ContactData to a TeamContact.
func MakeTeamContact(contact *moira.ContactData) TeamContact {
	return TeamContact{
		ID:           contact.ID,
		Name:         contact.Name,
		User:         contact.User,
		TeamID:       contact.Team,
		Team:         contact.Team,
		Type:         contact.Type,
		Value:        contact.Value,
		ExtraMessage: contact.ExtraMessage,
	}
}

type TeamContactWithScore struct {
	TeamContact

	Score *ContactScore `json:"score,omitempty" extensions:"x-nullable"`
}

// Render is a function that implements chi Renderer interface for TeamContact.
func (TeamContact) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

// TeamsList is a structure that represents a list of existing teams in db.
type TeamsList struct {
	List  []TeamModel `json:"list" binding:"required"`
	Page  int64       `json:"page" binding:"required" example:"0" format:"int64"`
	Size  int64       `json:"size" binding:"required" example:"100" format:"int64"`
	Total int64       `json:"total" binding:"required" example:"10" format:"int64"`
}

// Render is a function that implements chi Renderer interface for TeamsList.
func (TeamsList) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

// NewTeamsList constructs TeamsList out of []moira.Team.
// TeamsList.Page, TeamsList.Size and TeamsList.Total are not filled.
func NewTeamsList(teams []moira.Team) TeamsList {
	models := make([]TeamModel, 0, len(teams))

	for _, team := range teams {
		models = append(models, NewTeamModel(team))
	}

	return TeamsList{
		List: models,
	}
}
