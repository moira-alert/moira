package redis

import (
	"fmt"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database/redis/reply"
)

// SaveTeam saves team into redis
func (connector *DbConnector) SaveTeam(teamID string, team moira.Team) error {
	c := *connector.client

	teamBytes, err := reply.MarshallTeam(team)
	if err != nil {
		return fmt.Errorf("save team error: %w", err)
	}

	err = c.HSet(connector.context, teamsKey, teamID, teamBytes).Err()
	if err != nil {
		return fmt.Errorf("save team redis error: %w", err)
	}
	return nil
}

// GetTeam retrieves team from redis by it's id
func (connector *DbConnector) GetTeam(teamID string) (moira.Team, error) {
	c := *connector.client

	response := c.HGet(connector.context, teamsKey, teamID)
	team, err := reply.NewTeam(response)
	if err != nil {
		return moira.Team{}, err
	}
	team.ID = teamID

	return team, nil
}

// SaveTeamsAndUsers is a function that saves users for one team and teams for bunch of users in one transaction.
func (connector *DbConnector) SaveTeamsAndUsers(teamID string, users []string, teams map[string][]string) error {
	c := *connector.client

	pipe := c.TxPipeline()
	err := pipe.Del(connector.context, teamUsersKey(teamID)).Err()
	if err != nil {
		return fmt.Errorf("cannot clear users set for team: %s, %w", teamID, err)
	}
	for _, userID := range users {
		err = pipe.SAdd(connector.context, teamUsersKey(teamID), userID).Err()
		if err != nil {
			return fmt.Errorf("cannot save users for team: %s, %w", teamID, err)
		}
	}

	for userID, userTeams := range teams {
		err = pipe.Del(connector.context, userTeamsKey(userID)).Err()
		if err != nil {
			return fmt.Errorf("cannot clear teams set for user: %s, %w", userID, err)
		}
		for _, teamID := range userTeams {
			err = pipe.SAdd(connector.context, userTeamsKey(userID), teamID).Err()
			if err != nil {
				return fmt.Errorf("cannot save teams for user: %s, %w", userID, err)
			}
		}
	}

	_, err = pipe.Exec(connector.context)
	if err != nil {
		return fmt.Errorf("cannot commit transaction and save team: %w", err)
	}

	return nil
}

func (connector *DbConnector) GetUserTeams(userID string) ([]string, error) {
	c := *connector.client

	teams, err := c.SMembers(connector.context, userTeamsKey(userID)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve user teams: %w", err)
	}
	return teams, nil
}

// GetTeamUsers returns all users of certain team
func (connector *DbConnector) GetTeamUsers(teamID string) ([]string, error) {
	c := *connector.client

	teams, err := c.SMembers(connector.context, teamUsersKey(teamID)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve user teams: %w", err)
	}
	return teams, nil
}

// IsTeamContainUser is a method to check if user is in team.
func (connector *DbConnector) IsTeamContainUser(teamID, userID string) (bool, error) {
	c := *connector.client

	result, err := c.SIsMember(connector.context, teamUsersKey(teamID), userID).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check if team contains user: %w", err)
	}
	return result, nil
}

// DeleteTeam is a method to delete all information about team and remove team from last user's teams.
func (connector *DbConnector) DeleteTeam(teamID, userID string) error {
	c := *connector.client

	pipe := c.TxPipeline()

	err := pipe.SRem(connector.context, userTeamsKey(userID), teamID).Err()
	if err != nil {
		return fmt.Errorf("failed to remove team from user's teams: %w", err)
	}

	err = pipe.Del(connector.context, teamUsersKey(teamID)).Err()
	if err != nil {
		return fmt.Errorf("failed to remove team users: %w", err)
	}

	err = pipe.HDel(connector.context, teamsKey, teamID).Err()
	if err != nil {
		return fmt.Errorf("failed to remove team metadata: %w", err)
	}

	_, err = pipe.Exec(connector.context)
	if err != nil {
		return fmt.Errorf("cannot commit transaction and delete team: %w", err)
	}

	return nil
}

// GetAllTeams returns all teams.
func (connector *DbConnector) GetAllTeams() ([]*moira.Team, error) {
	c := *connector.client
	cmd := c.HGetAll(connector.context, teamsKey)
	teams, err := reply.NewTeams(cmd)
	if err != nil {
		return nil, err
	}

	return teams, nil
}

const teamsKey = "moira-teams"

func userTeamsKey(userID string) string {
	return fmt.Sprintf("moira-userTeams:%s", userID)
}

func teamUsersKey(teamID string) string {
	return fmt.Sprintf("moira-teamUsers:%s", teamID)
}
