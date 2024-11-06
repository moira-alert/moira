package redis

import (
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/moira-alert/moira/database"
	"strings"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database/redis/reply"
)

// SaveTeam saves team into redis.
func (connector *DbConnector) SaveTeam(teamID string, team moira.Team) error {
	c := *connector.client

	newTeamLowercaseName := strings.ToLower(team.Name)

	teamBytes, err := reply.MarshallTeam(team)
	if err != nil {
		return fmt.Errorf("failed to marshal team: %w", err)
	}

	// need to use watch here because if team name is updated
	// we also need to change name in moira-teams-names set
	err = c.Watch(
		connector.context,
		func(tx *redis.Tx) error {
			nameExists, err := tx.SIsMember(connector.context, teamsNamesKey, newTeamLowercaseName).Result()
			if err != nil {
				return fmt.Errorf("failed to check team name existance: %w", err)
			}

			if nameExists {
				return database.ErrTeamWithNameAlreadyExists
			}

			// try to get team with such id
			response := tx.HGet(connector.context, teamsKey, teamID)
			existedTeam, err := reply.NewTeam(response)
			if err != nil && !errors.Is(err, database.ErrNil) {
				return fmt.Errorf("failed to get team: %w", err)
			}

			pipe := tx.TxPipeline()
			existedTeamLowercaseName := strings.ToLower(existedTeam.Name)

			// if team already exists and team.Name is changed we should delete previous name
			// from moira-teams-names set.
			if err == nil && existedTeamLowercaseName != newTeamLowercaseName {
				err = pipe.SRem(connector.context, teamsNamesKey, existedTeamLowercaseName).Err()
				if err != nil {
					return fmt.Errorf("failed to update team name: %w", err)
				}
			}

			// save team
			err = pipe.HSet(connector.context, teamsKey, teamID, teamBytes).Err()
			if err != nil {
				return fmt.Errorf("failed to save team metadata: %w", err)
			}

			// save team name
			err = pipe.SAdd(connector.context, teamsNamesKey, newTeamLowercaseName).Err()
			if err != nil {
				return fmt.Errorf("failed to save team name: %w", err)
			}

			_, err = pipe.Exec(connector.context)
			if err != nil {
				return fmt.Errorf("cannot commit transaction and save team: %w", err)
			}

			return nil
		},
		teamsNamesKey)

	return err
}

// GetTeam retrieves team from redis by it's id.
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

// GetTeamUsers returns all users of certain team.
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
	team, err := connector.GetTeam(teamID)
	if err != nil {
		return fmt.Errorf("failed to get team to delete: %w", err)
	}

	c := *connector.client

	pipe := c.TxPipeline()

	err = pipe.SRem(connector.context, userTeamsKey(userID), teamID).Err()
	if err != nil {
		return fmt.Errorf("failed to remove team from user's teams: %w", err)
	}

	err = pipe.Del(connector.context, teamUsersKey(teamID)).Err()
	if err != nil {
		return fmt.Errorf("failed to remove team users: %w", err)
	}

	err = pipe.SRem(connector.context, teamsNamesKey, strings.ToLower(team.Name)).Err()
	if err != nil {
		return fmt.Errorf("failed to remove team name: %w", err)
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

const (
	teamsKey      = "moira-teams"
	teamsNamesKey = "moira-teams-names"
)

func userTeamsKey(userID string) string {
	return fmt.Sprintf("moira-userTeams:%s", userID)
}

func teamUsersKey(teamID string) string {
	return fmt.Sprintf("moira-teamUsers:%s", teamID)
}
