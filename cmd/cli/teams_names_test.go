package main

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/moira-alert/moira/database/redis"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	"github.com/stretchr/testify/require"
)

var testTeams = []teamWithID{
	{
		teamStorageElement: teamStorageElement{
			Name:        "First team",
			Description: "first desc",
		},
		ID: "team1",
	},
	{
		teamStorageElement: teamStorageElement{
			Name:        "Second team",
			Description: "second desc",
		},
		ID: "team2",
	},
	{
		teamStorageElement: teamStorageElement{
			Name:        "Third team",
			Description: "third desc",
		},
		ID: "team3",
	},
	{
		teamStorageElement: teamStorageElement{
			Name:        "Fourth team",
			Description: "fourth desc",
		},
		ID: "team4",
	},
}

func Test_fillTeamNamesHash(t *testing.T) {
	conf := getDefault()

	logger, err := logging.ConfigureLog(conf.LogFile, conf.LogLevel, "test", conf.LogPrettyFormat)
	require.NoError(t, err)

	db := redis.NewTestDatabase(logger)

	db.Flush()
	defer db.Flush()

	ctx := context.Background()
	client := db.Client()

	t.Run("with empty database", func(t *testing.T) {
		err = fillTeamNamesHash(logger, db)
		require.NoError(t, err)

		res, existErr := client.Exists(ctx, teamsByNamesKey).Result()
		require.NoError(t, existErr)
		require.Equal(t, int64(0), res)
	})

	t.Run("with teams which have unique names", func(t *testing.T) {
		defer db.Flush()

		teamNames := make(map[string]string, len(testTeams))

		for _, team := range testTeams {
			teamBytes, err := getTeamBytes(team)
			require.NoError(t, err)

			err = client.HSet(ctx, teamsKey, team.ID, teamBytes).Err()
			require.NoError(t, err)

			teamNames[strings.ToLower(team.Name)] = team.ID
		}

		err = fillTeamNamesHash(logger, db)
		require.NoError(t, err)

		actualTeamNames, err := client.HGetAll(ctx, teamsByNamesKey).Result()
		require.NoError(t, err)
		require.Equal(t, teamNames, actualTeamNames)
	})

	t.Run("with teams no unique names", func(t *testing.T) {
		defer db.Flush()

		testTeams[0].Name = "Team name"
		testTeams[1].Name = "teaM name"
		testTeams[2].Name = "Team name"

		for _, team := range testTeams {
			teamBytes, err := getTeamBytes(team)
			require.NoError(t, err)

			err = client.HSet(ctx, teamsKey, team.ID, teamBytes).Err()
			require.NoError(t, err)
		}

		err = fillTeamNamesHash(logger, db)
		require.NoError(t, err)

		actualTeamNames, err := client.HGetAll(ctx, teamsByNamesKey).Result()
		require.NoError(t, err)
		require.Len(t, actualTeamNames, len(testTeams))

		expectedLowercasedTeamNames := []string{"team name", "team name1", "team name2", strings.ToLower(testTeams[3].Name)}
		for _, name := range expectedLowercasedTeamNames {
			_, ok := actualTeamNames[name]
			require.True(t, ok)
		}

		for i, team := range testTeams {
			t.Run(fmt.Sprintf("for team %v fields ok", i), func(t *testing.T) {
				marshaledTeam, err := client.HGet(ctx, teamsKey, team.ID).Result()
				require.NoError(t, err)

				actualTeam, err := unmarshalTeam(team.ID, []byte(marshaledTeam))
				require.NoError(t, err)
				require.Equal(t, team.ID, actualTeam.ID)
				require.Equal(t, team.Description, actualTeam.Description)

				if i < 3 {
					require.Contains(t, []string{team.Name, team.Name + "1", team.Name + "2"}, actualTeam.Name)
				} else {
					require.Equal(t, team.Name, actualTeam.Name)
				}
			})
		}
	})

	t.Run("with teams has no unique names and adding one number does not help", func(t *testing.T) {
		testTeams[0].Name = "Team name"
		testTeams[1].Name = "teaM name"
		testTeams[2].Name = "Team name1"

		for _, team := range testTeams {
			teamBytes, err := getTeamBytes(team)
			require.NoError(t, err)

			err = client.HSet(ctx, teamsKey, team.ID, teamBytes).Err()
			require.NoError(t, err)
		}

		err = fillTeamNamesHash(logger, db)
		require.NoError(t, err)

		actualTeamNames, err := client.HGetAll(ctx, teamsByNamesKey).Result()
		require.NoError(t, err)
		require.Len(t, actualTeamNames, len(testTeams))

		expectedLowercasedTeamNames := []string{"team name", "team name1", "team name1_0", "team name1_1", strings.ToLower(testTeams[3].Name)}
		for name := range actualTeamNames {
			require.Contains(t, expectedLowercasedTeamNames, name)
		}

		for i, team := range testTeams {
			t.Run(fmt.Sprintf("for team %v fields ok", i), func(t *testing.T) {
				marshaledTeam, err := client.HGet(ctx, teamsKey, team.ID).Result()
				require.NoError(t, err)

				actualTeam, err := unmarshalTeam(team.ID, []byte(marshaledTeam))
				require.NoError(t, err)
				require.Equal(t, team.ID, actualTeam.ID)
				require.Equal(t, team.Description, actualTeam.Description)

				if i < 3 {
					require.Contains(t, []string{team.Name, team.Name + "1", team.Name + "1_1", team.Name + "_0"}, actualTeam.Name)
				} else {
					require.Equal(t, team.Name, actualTeam.Name)
				}
			})
		}
	})
}

func Test_removeTeamNamesHash(t *testing.T) {
	conf := getDefault()

	logger, err := logging.ConfigureLog(conf.LogFile, conf.LogLevel, "test", conf.LogPrettyFormat)
	require.NoError(t, err)

	db := redis.NewTestDatabase(logger)

	db.Flush()
	defer db.Flush()

	ctx := context.Background()
	client := db.Client()

	t.Run("with empty database", func(t *testing.T) {
		err = removeTeamNamesHash(logger, db)
		require.NoError(t, err)

		res, existErr := client.Exists(ctx, teamsByNamesKey).Result()
		require.NoError(t, existErr)
		require.Equal(t, int64(0), res)
	})

	t.Run("with filled teams and teams by names hashes", func(t *testing.T) {
		defer db.Flush()

		for _, team := range testTeams {
			teamBytes, err := getTeamBytes(team)
			require.NoError(t, err)

			err = client.HSet(ctx, teamsKey, team.ID, teamBytes).Err()
			require.NoError(t, err)

			err = client.HSet(ctx, teamsByNamesKey, strings.ToLower(team.Name), team.ID).Err()
			require.NoError(t, err)
		}

		err = removeTeamNamesHash(logger, db)
		require.NoError(t, err)

		res, existErr := client.Exists(ctx, teamsByNamesKey).Result()
		require.NoError(t, existErr)
		require.Equal(t, int64(0), res)
	})
}
