package redis

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-redis/redis/v8"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database"
	"github.com/moira-alert/moira/database/redis/reply"
)

const teamSaveDeleteAttempts = 3

// SaveTeam saves team into redis.
func (connector *DbConnector) SaveTeam(teamID string, team moira.Team) error {
	c := *connector.client

	newTeamLowercaseName := strings.ToLower(team.Name)

	teamBytes, err := reply.MarshallTeam(team)
	if err != nil {
		return fmt.Errorf("failed to marshal team: %w", err)
	}

	saveTeamInTx := func(tx *redis.Tx) error {
		teamWithNameExists, err := connector.isTeamExist(tx, team.Name)
		if err != nil {
			return err
		}

		// try to get team with such id
		existedTeam, err := connector.getTeamInTx(tx, teamID)
		if err != nil && !errors.Is(err, database.ErrNil) {
			return fmt.Errorf("failed to get team: %w", err)
		}

		// team with such id does not exist but another team with such name exists
		if err != nil && teamWithNameExists {
			return database.ErrTeamWithNameAlreadyExists
		}

		existedTeamLowercaseName := strings.ToLower(existedTeam.Name)

		_, err = tx.TxPipelined(
			connector.context,
			func(pipe redis.Pipeliner) error {
				updateTeamName := err == nil && existedTeamLowercaseName != newTeamLowercaseName

				// if team with such id already exists and team.Name is changed
				if updateTeamName {
					// but team with new name already exists
					if teamWithNameExists {
						return database.ErrTeamWithNameAlreadyExists
					}

					// remove old team.Name from team names set
					err = pipe.HDel(connector.context, teamsByNamesKey, existedTeamLowercaseName).Err()
					if err != nil {
						return fmt.Errorf("failed to update team name: %w", err)
					}
				}

				// save team
				err = pipe.HSet(connector.context, teamsKey, teamID, teamBytes).Err()
				if err != nil {
					return fmt.Errorf("failed to save team metadata: %w", err)
				}

				err = pipe.HSet(connector.context, teamsByNamesKey, newTeamLowercaseName, teamID).Err()
				if err != nil {
					return fmt.Errorf("failed to save team name: %w", err)
				}

				return nil
			})

		return err
	}

	for i := 1; i < teamSaveDeleteAttempts; i++ {
		// need to use watch here because if team name is updated
		// we also need to change name in moira-teams-names set
		err = c.Watch(connector.context, saveTeamInTx, teamsByNamesKey)

		if err == nil {
			return nil
		}

		if !errors.Is(err, redis.TxFailedErr) {
			return err
		}
	}

	return err
}

func (connector *DbConnector) getTeamInTx(tx *redis.Tx, teamID string) (moira.Team, error) {
	response := tx.HGet(connector.context, teamsKey, teamID)
	team, err := reply.NewTeam(response)
	if err != nil {
		return moira.Team{}, err
	}
	team.ID = teamID

	return team, nil
}

func (connector *DbConnector) isTeamExist(tx *redis.Tx, teamName string) (bool, error) {
	nameExists, err := tx.HExists(connector.context, teamsByNamesKey, strings.ToLower(teamName)).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check team name existance: %w", err)
	}

	return nameExists, nil
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

// GetTeamByName retrieves team from redis by its name. Note that
func (connector *DbConnector) GetTeamByName(name string) (moira.Team, error) {
	c := *connector.client

	teamID, err := c.HGet(connector.context, teamsByNamesKey, strings.ToLower(name)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return moira.Team{}, database.ErrNil
		}

		return moira.Team{}, fmt.Errorf("failed to get team by name: %w", err)
	}

	return connector.GetTeam(teamID)
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
	c := *connector.client

	deleteTeamInTx := func(tx *redis.Tx) error {
		team, err := connector.getTeamInTx(tx, teamID)
		if err != nil {
			return fmt.Errorf("failed to get team to delete: %w", err)
		}

		_, err = tx.TxPipelined(
			connector.context,
			func(pipe redis.Pipeliner) error {
				err = pipe.SRem(connector.context, userTeamsKey(userID), teamID).Err()
				if err != nil {
					return fmt.Errorf("failed to remove team from user's teams: %w", err)
				}

				err = pipe.Del(connector.context, teamUsersKey(teamID)).Err()
				if err != nil {
					return fmt.Errorf("failed to remove team users: %w", err)
				}

				err = pipe.HDel(connector.context, teamsByNamesKey, strings.ToLower(team.Name)).Err()
				if err != nil {
					return fmt.Errorf("failed to remove team name: %w", err)
				}

				err = pipe.HDel(connector.context, teamsKey, teamID).Err()
				if err != nil {
					return fmt.Errorf("failed to remove team metadata: %w", err)
				}

				return nil
			})

		return err
	}

	var err error
	for i := 1; i < teamSaveDeleteAttempts; i++ {
		// need to use watch here because if team is deleted
		// we also need to remove team name
		err = c.Watch(connector.context, deleteTeamInTx, teamsByNamesKey)

		if err == nil {
			return nil
		}

		if !errors.Is(err, redis.TxFailedErr) {
			return err
		}
	}

	return err
}

const (
	teamsKey        = "moira-teams"
	teamsByNamesKey = "moira-teams-by-names"
)

func userTeamsKey(userID string) string {
	return fmt.Sprintf("moira-userTeams:%s", userID)
}

func teamUsersKey(teamID string) string {
	return fmt.Sprintf("moira-teamUsers:%s", teamID)
}
