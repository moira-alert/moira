package redis

import (
	"errors"

	"github.com/moira-alert/moira"
)

func (connector *DbConnector) SaveTeam(teamID string, team moira.Team) error {
	return errors.New("not implemented")
}

func (connector *DbConnector) GetTeam(teamID string) (moira.Team, error) {
	return moira.Team{}, errors.New("not implemented")
}
func (connector *DbConnector) SaveTeamsAndUsers(teamID string, users []string, usersTeams map[string][]string) error {
	return errors.New("not implemented")
}

func (connector *DbConnector) GetUserTeams(userID string) ([]string, error) {
	return []string{}, errors.New("not implemented")
}
func (connector *DbConnector) GetTeamUsers(teamID string) ([]string, error) {
	return []string{}, errors.New("not implemented")
}
