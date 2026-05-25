package postgresql

/*
Scheme: teams -> team_members <- users
*/

import (
	"fmt"

	"github.com/moira-alert/moira"
)

// SaveTeam saves team into postgresql.
func (connector *DbConnector) SaveTeam(teamID string, team moira.Team) error {
	query := `
INSERT INTO teams (team_id, name, description)
VALUES ($1, $2, $3);`
	_, err := connector.db.Master().ExecContext(connector.ctx, query,
		teamID,
		team.Name,
		team.Description,
	)
	if err != nil {
		return fmt.Errorf("save team error: %w", err)
	}
	return nil
}

func (connector *DbConnector) GetAllTeams() ([]moira.Team, error) {
	query := "SELECT team_id, name, description FROM teams;"
	responseFromReplica, err := connector.db.Replica().QueryContext(connector.ctx, query)
	if err != nil {
		return nil, fmt.Errorf("get all teams error: %w", err)
	}

	teams := make([]moira.Team, 0)
	for responseFromReplica.Next() {
		var (
			id string
			name string
			description string
		)
		err := responseFromReplica.Scan(&id, &name, &description)
		if err != nil {
			return nil, fmt.Errorf("unmarshaling error on get all teams: %w", err)
		}
		teams = append(teams, moira.Team{
			ID: id,
			Name: name,
			Description: description,
		})
	}
	return teams, nil
}

func (connector *DbConnector) GetTeam(teamID string) (moira.Team, error) {
	query := "SELECT id, name, description FROM teams WHERE id=$1 LIMIT 1;"
	requester := func(db RDB) (moira.Team, error) {
		row := db.QueryRowContext(connector.ctx, query, teamID)
		var (
			id string
			name string
			description string
		)
		err := row.Scan(&id, &name, &description)
		return moira.Team{
			ID: id,
			Name: name,
			Description: description,
		}, err
	}

	respFromReplica, err := requester(connector.db.Replica())
	if err == nil {
		return respFromReplica, nil
	}

	respFromMaster, err := requester(connector.db.Master())
	return respFromMaster, err
}

func (connector *DbConnector) GetTeamByName(name string) (moira.Team, error) {
	query := "SELECT id, name, description FROM teams WHERE name=$1 LIMIT 1;"
	requester := func(db RDB) (moira.Team, error) {
		row := db.QueryRowContext(connector.ctx, query, name)
		var (
			id string
			name string
			description string
		)
		err := row.Scan(&id, &name, &description)
		return moira.Team{
			ID: id,
			Name: name,
			Description: description,
		}, err
	}

	respFromReplica, err := requester(connector.db.Replica())
	if err == nil {
		return respFromReplica, nil
	}

	respFromMaster, err := requester(connector.db.Master())
	return respFromMaster, err
}

func (connector *DbConnector) SaveTeamsAndUsers(teamID string, users []string, teams map[string][]string) error {
	clearQuery := `
DELETE FROM team_members WHERE team_id = $1
	`
	insertQuery := `
INSERT INTO team_members (team_id, user_id) VALUES ($1, $2)
	`

	//TODO: interact via transactions manager
	if _, err := connector.db.Master().ExecContext(connector.ctx, clearQuery, teamID); err != nil {
		return fmt.Errorf("clear team members error: %w", err)
	}

	for _, userID := range users {
		if _, err := connector.db.Master().ExecContext(connector.ctx, insertQuery, teamID, userID); err != nil {
			return fmt.Errorf("error on save user %s to group %s: %w", userID, teamID, err)
		}
	}

	return nil
}

func (connector *DbConnector) GetTeamUsers(teamID string) ([]string, error) {
	query := `
SELECT user_id FROM team_members WHERE team_id = $1
	`

	responseFromReplica, err := connector.db.Replica().QueryContext(connector.ctx, query)
	if err != nil {
		return nil, fmt.Errorf("get users of team %s error: %w", teamID, err)
	}

	users := make([]string, 0)
	for responseFromReplica.Next() {
		var userId string
		err := responseFromReplica.Scan(&userId)
		if err != nil {
			return nil, fmt.Errorf("unmarshaling error on get user: %w", err)
		}
		users = append(users, userId)
	}
	return users, nil
}

func (connector *DbConnector) IsTeamContainUser(teamID, userID string) (bool, error) {
	query := `
SELECT EXISTS (
	SELECT 1 FROM team_members WHERE team_id = $1 AND user_id = $2
)
	`
	responseFromReplica, err := connector.db.Replica().QueryContext(connector.ctx, query)
	if err != nil {
		return false, fmt.Errorf("check user %s existance in team %s error: %w", userID, teamID, err)
	}

	var exists bool
	if err := responseFromReplica.Scan(&exists); err != nil {
		return false, fmt.Errorf("unmarshaling error on check user existance: %w", err)
	}

	return exists, nil
}

func (connector *DbConnector) DeleteTeam(teamID, userID string) error {
	query := `
DELETE FROM teams WHERE team_id = $1
	`
	if _, err := connector.db.Master().ExecContext(connector.ctx, query, teamID); err != nil {
		return fmt.Errorf("clear team members error: %w", err)
	}

	return nil
}
