package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

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
//  5. Save pairs teamName:team.ID to "moira-teams-by-names" redis hash.
func fillTeamNamesHash(logger moira.Logger, database moira.Database) error {
	logger.Info().Msg("Start filling \"moira-teams-by-names\" hash")

	switch db := database.(type) {
	case *redis.DbConnector:
		logger.Info().Msg("collecting teams from redis node...")

		teamsMap, err := db.Client().HGetAll(db.Context(), teamsKey).Result()
		if err != nil {
			return fmt.Errorf("failed to fetch teams from redis node: %w", err)
		}

		logger.Info().
			Int("total_teams_count", len(teamsMap)).
			Msg("fetched teams")

		teamsByNameMap, err := groupTeamsByNames(logger, teamsMap)
		if err != nil {
			return fmt.Errorf("failed to group teams by names: %w", err)
		}

		teamByUniqueName := transformTeamsByNameMap(teamsByNameMap)

		client := db.Client()
		ctx := db.Context()

		_, pipeErr := client.TxPipelined(
			ctx,
			func(pipe goredis.Pipeliner) error {
				return updateTeamsInPipe(ctx, logger, pipe, teamByUniqueName)
			})
		if pipeErr != nil {
			return pipeErr
		}

	default:
		return makeUnknownDBError(database)
	}
	return nil
}

func groupTeamsByNames(logger moira.Logger, teamsMap map[string]string) (map[string][]teamWithID, error) {
	teamsByNameMap := make(map[string][]teamWithID, len(teamsMap))

	for teamID, marshaledTeam := range teamsMap {
		team, err := unmarshalTeam(teamID, []byte(marshaledTeam))
		if err != nil {
			return nil, err
		}

		lowercaseTeamName := strings.ToLower(team.Name)

		teamWithNameList, exists := teamsByNameMap[lowercaseTeamName]
		if exists {
			teamWithNameList = append(teamWithNameList, team)
			teamsByNameMap[lowercaseTeamName] = teamWithNameList
		} else {
			teamsByNameMap[lowercaseTeamName] = []teamWithID{team}
		}
	}

	logger.Info().
		Int("unique_team_names_count", len(teamsByNameMap)).
		Msg("grouped teams with same names")

	return teamsByNameMap, nil
}

func transformTeamsByNameMap(teamsByNameMap map[string][]teamWithID) map[string]teamWithID {
	teamByUniqueName := make(map[string]teamWithID, len(teamsByNameMap))

	for _, teams := range teamsByNameMap {
		for i, team := range teams {
			iStr := strconv.FormatInt(int64(i), 10)

			if i > 0 {
				team.Name += iStr
			}

			for {
				// sometimes we have the following situation in db (IDs and team names):
				// moira-teams: {
				//    team1: "team name",
				//    team2: "team Name",
				//    team3: "Team name1"
				// }
				// so we can't just add 1 to one of [team1, team2]
				lowercasedTeamName := strings.ToLower(team.Name)

				_, exists := teamByUniqueName[lowercasedTeamName]
				if exists {
					team.Name += "_" + iStr
				} else {
					teamByUniqueName[lowercasedTeamName] = team
					break
				}
			}
		}
	}

	return teamByUniqueName
}

func updateTeamsInPipe(ctx context.Context, logger moira.Logger, pipe goredis.Pipeliner, teamsByUniqueName map[string]teamWithID) error {
	for _, team := range teamsByUniqueName {
		teamBytes, err := getTeamBytes(team)
		if err != nil {
			return err
		}

		err = pipe.HSet(ctx, teamsKey, team.ID, teamBytes).Err()
		if err != nil {
			logger.Error().
				Error(err).
				String("team_id", team.ID).
				String("new_team_name", team.Name).
				Msg("failed to update team name")

			return fmt.Errorf("failed to update team name: %w", err)
		}

		err = pipe.HSet(ctx, teamsByNamesKey, strings.ToLower(team.Name), team.ID).Err()
		if err != nil {
			logger.Error().
				Error(err).
				String("team_id", team.ID).
				String("new_team_name", team.Name).
				Msg("failed to add team name to redis hash")

			return fmt.Errorf("failed to add team name to redis hash: %w", err)
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
		_, err := db.Client().Del(db.Context(), teamsByNamesKey).Result()
		if err != nil {
			return fmt.Errorf("failed to delete teamsByNameKey: %w", err)
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
