package reply

import (
	"encoding/json"
	"fmt"

	"github.com/gomodule/redigo/redis"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database"
)

// teamStorageElement is a representation of team in database
type teamStorageElement struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func newTeamStorageElement(team moira.Team) teamStorageElement {
	return teamStorageElement{
		Name:        team.Name,
		Description: team.Description,
	}
}

func (t *teamStorageElement) toTeam() moira.Team {
	return moira.Team{
		Name:        t.Name,
		Description: t.Description,
	}
}

// MarshallTeam is a function that converts team to the bytes that can be held in database
func MarshallTeam(team moira.Team) ([]byte, error) {
	teamSE := newTeamStorageElement(team)
	bytes, err := json.Marshal(teamSE)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal team: %w", err)
	}
	return bytes, nil
}

// NewTeam is a function that creates a team entity from a bytes received from database
func NewTeam(rep interface{}, err error) (moira.Team, error) {
	bytes, err := redis.Bytes(rep, err)
	if err != nil {
		if err == redis.ErrNil {
			return moira.Team{}, database.ErrNil
		}
		return moira.Team{}, fmt.Errorf("failed to read team: %w", err)
	}
	teamSE := teamStorageElement{}
	err = json.Unmarshal(bytes, &teamSE)
	if err != nil {
		return moira.Team{}, fmt.Errorf("failed to parse team json %s: %w", string(bytes), err)
	}
	return teamSE.toTeam(), nil
}
