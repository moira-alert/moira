package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"

	goredis "github.com/go-redis/redis/v8"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database/redis"
)

const (
	teamsKey        = "moira-teams"
	teamsByNamesKey = "moira-teams-by-names"
)

// fillTeamNamesHash does the following
//  1. Get all teams from DB.
//  2. Group teams with same names.
//  3. For teams with same names change name (example: ["name", "name", "name"] -> ["name", "name1", "name2"]).
//  4. Update teams with name changed.
//  5. Save pair teamName:team.ID to "moira-teams-by-names" redis hash.
func fillTeamNamesHash(logger moira.Logger, database moira.Database) error {
	logger.Info().Msg("Start filling \"moira-teams-by-names\" hash")

	switch db := database.(type) {
	case *redis.DbConnector:
		teamMaps := make([]map[string]string, 0)
		mutex := sync.Mutex{}

		err := callFunc(
			db,
			func(connector *redis.DbConnector, client goredis.UniversalClient) error {
				ctx := connector.Context()

				logger.Info().Msg("collecting teams from redis node...")

				resMap, err := client.HGetAll(ctx, teamsKey).Result()
				if err != nil {
					return fmt.Errorf("failed to fetch teams from redis node: %w", err)
				}

				// func passed to callFunc may be called concurrently
				mutex.Lock()
				defer mutex.Unlock()

				teamMaps = append(teamMaps, resMap)

				return nil
			})
		if err != nil {
			return err
		}

		teamsByNameMap, err := groupTeamsByNames(logger, teamMaps)
		if err != nil {
			return fmt.Errorf("failed to group teams by names: %w", err)
		}

		client := db.Client()
		ctx := db.Context()

		_, pipeErr := client.TxPipelined(
			ctx,
			func(pipe goredis.Pipeliner) error {
				return updateTeamsInPipe(ctx, logger, pipe, teamsByNameMap)
			})
		if pipeErr != nil {
			return pipeErr
		}

	default:
		return makeUnknownDBError(database)
	}
	return nil
}

func groupTeamsByNames(logger moira.Logger, teamMaps []map[string]string) (map[string][]teamWithID, error) {
	mapSize := 0
	for _, m := range teamMaps {
		collectedFromNode := len(m)

		logger.Info().
			Int("teams_count", collectedFromNode).
			Msg("successfully collected from redis node")

		mapSize += len(m)
	}

	logger.Info().
		Int("total_teams_count", mapSize).
		Msg("from all nodes")

	teamsByNameMap := make(map[string][]teamWithID, mapSize)

	for _, m := range teamMaps {
		for teamID, marshaledTeam := range m {
			team, err := unmarshalTeam(teamID, []byte(marshaledTeam))
			if err != nil {
				return nil, err
			}

			teamWithNameList, exists := teamsByNameMap[team.Name]
			if exists {
				teamWithNameList = append(teamWithNameList, team)
				teamsByNameMap[team.Name] = teamWithNameList
			} else {
				teamsByNameMap[team.Name] = []teamWithID{team}
			}
		}
	}

	logger.Info().
		Int("unique_team_names_count", len(teamsByNameMap)).
		Msg("grouped teams with same names")

	return teamsByNameMap, nil
}

func updateTeamsInPipe(ctx context.Context, logger moira.Logger, pipe goredis.Pipeliner, teamsByNameMap map[string][]teamWithID) error {
	for teamName, teams := range teamsByNameMap {
		for i, team := range teams {
			if i > 0 {
				// there more than 1 team with same name, so updating teams by adding digit to the name end
				team.Name += strconv.FormatInt(int64(i), 10)

				teamBytes, err := getTeamBytes(team)
				if err != nil {
					return err
				}

				err = pipe.HSet(ctx, teamsKey, team.ID, teamBytes).Err()
				if err != nil {
					logger.Error().
						Error(err).
						String("team_id", team.ID).
						String("prev_team_name", teamName).
						String("new_team_name", team.Name).
						Msg("failed to update team name")

					return fmt.Errorf("failed to update team name: %w", err)
				}
			}

			err := pipe.HSet(ctx, teamsByNamesKey, strings.ToLower(team.Name), team.ID).Err()
			if err != nil {
				logger.Error().
					Error(err).
					String("team_id", team.ID).
					String("prev_team_name", teamName).
					String("new_team_name", team.Name).
					Msg("failed to add team name to redis hash")

				return fmt.Errorf("failed to add team name to redis hash: %w", err)
			}
		}
	}

	return nil
}

// removeTeamNamesHash remove "moira-teams-by-names" redis hash.
// Note that if fillTeamNamesHash have been called, then team names would not be changed back.
func removeTeamNamesHash(logger moira.Logger, database moira.Database) error {
	logger.Info().Msg("Start removing \"moira-teams-by-names\" redis hash")

	switch db := database.(type) {
	case *redis.DbConnector:
		err := callFunc(
			db,
			func(connector *redis.DbConnector, client goredis.UniversalClient) error {
				_, err := client.Del(connector.Context(), teamsByNamesKey).Result()
				if err != nil {
					return fmt.Errorf("failed to delete teamsByNameKey: %w", err)
				}

				return nil
			})
		if err != nil {
			return err
		}

	default:
		return makeUnknownDBError(database)
	}

	return nil
}

type teamStorageElement struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type teamWithID struct {
	teamStorageElement
	ID string
}

func unmarshalTeam(teamID string, teamBytes []byte) (teamWithID, error) {
	var storedTeam teamStorageElement
	err := json.Unmarshal(teamBytes, &storedTeam)
	if err != nil {
		return teamWithID{}, fmt.Errorf("failed to deserialize team: %w", err)
	}

	return teamWithID{
		teamStorageElement: storedTeam,
		ID:                 teamID,
	}, nil
}

func getTeamBytes(team teamWithID) ([]byte, error) {
	bytes, err := json.Marshal(team.teamStorageElement)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal team: %w", err)
	}

	return bytes, nil
}
