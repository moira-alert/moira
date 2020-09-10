package redis

import (
	"fmt"

	"github.com/gomodule/redigo/redis"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database/redis/reply"
)

// SaveTeam saves team into redis
func (connector *DbConnector) SaveTeam(teamID string, team moira.Team) error {
	c := connector.pool.Get()
	defer c.Close()

	teamBytes, err := reply.MarshallTeam(team)
	if err != nil {
		return fmt.Errorf("save team error: %w", err)
	}

	_, err = c.Do("HSET", teamsKey, teamID, teamBytes)
	if err != nil {
		return fmt.Errorf("save team redis error: %w", err)
	}
	return nil
}

// GetTeam retrieves team from redis by it's id
func (connector *DbConnector) GetTeam(teamID string) (moira.Team, error) {
	c := connector.pool.Get()
	defer c.Close()

	response, err := c.Do("HGET", teamsKey, teamID)
	if err != nil {
		return moira.Team{}, fmt.Errorf("failed to retrieve team: %w", err)
	}

	team, err := reply.NewTeam(response, err)
	if err != nil {
		return moira.Team{}, err
	}
	team.ID = teamID

	return team, nil
}

// SaveTeamsAndUsers is a function that saves users for one team and teams for bunch of users in one transaction.
func (connector *DbConnector) SaveTeamsAndUsers(teamID string, users []string, teams map[string][]string) error {
	c := connector.pool.Get()
	defer c.Close()

	err := c.Send("MULTI")
	if err != nil {
		return fmt.Errorf("cannot open transaction %w", err)
	}

	for _, userID := range users {
		err = c.Send("SADD", teamUsersKey(teamID), userID)
		if err != nil {
			return fmt.Errorf("cannot save users for team: %s, %w", teamID, err)
		}
	}

	for userID, userTeams := range teams {
		for _, teamID := range userTeams {
			err = c.Send("SADD", userTeamsKey(userID), teamID)
			if err != nil {
				return fmt.Errorf("cannot save teams for user: %s, %w", userID, err)
			}
		}
	}

	_, err = c.Do("EXEC")
	if err != nil {
		return fmt.Errorf("cannot commit transaction and save team: %w", err)
	}

	return nil
}

func (connector *DbConnector) GetUserTeams(userID string) ([]string, error) {
	c := connector.pool.Get()
	defer c.Close()

	teams, err := redis.Strings(c.Do("SMEMBERS", userTeamsKey(userID)))
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve user teams: %w", err)
	}
	return teams, nil
}

// GetTeamUsers returns all users of certain team
func (connector *DbConnector) GetTeamUsers(teamID string) ([]string, error) {
	c := connector.pool.Get()
	defer c.Close()

	teams, err := redis.Strings(c.Do("SMEMBERS", teamUsersKey(teamID)))
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve user teams: %w", err)
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
