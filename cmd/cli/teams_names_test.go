package main

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/moira-alert/moira/database/redis"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"

	. "github.com/smartystreets/goconvey/convey"
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
	Convey("Test filling \"moira-teams-by-names\" redis hash", t, func() {
		conf := getDefault()

		logger, err := logging.ConfigureLog(conf.LogFile, conf.LogLevel, "test", conf.LogPrettyFormat)
		if err != nil {
			t.Fatal(err)
		}

		db := redis.NewTestDatabase(logger)

		db.Flush()
		defer db.Flush()

		ctx := context.Background()
		client := db.Client()

		Convey("with empty database", func() {
			err = fillTeamNamesHash(logger, db)
			So(err, ShouldBeNil)

			res, existErr := client.Exists(ctx, teamsByNamesKey).Result()
			So(existErr, ShouldBeNil)
			So(res, ShouldEqual, 0)
		})

		Convey("with teams which have unique names", func() {
			defer db.Flush()

			var teamNames, actualTeamNames map[string]string

			teamNames = make(map[string]string, len(testTeams))

			for _, team := range testTeams {
				var teamBytes []byte

				teamBytes, err = getTeamBytes(team)
				So(err, ShouldBeNil)

				err = client.HSet(ctx, teamsKey, team.ID, teamBytes).Err()
				So(err, ShouldBeNil)

				teamNames[strings.ToLower(team.Name)] = team.ID
			}

			err = fillTeamNamesHash(logger, db)
			So(err, ShouldBeNil)

			actualTeamNames, err = client.HGetAll(ctx, teamsByNamesKey).Result()
			So(err, ShouldBeNil)
			So(actualTeamNames, ShouldResemble, teamNames)
		})

		Convey("with teams no unique names", func() {
			defer db.Flush()

			testTeams[0].Name = "Team name"
			testTeams[1].Name = "teaM name"
			testTeams[2].Name = "Team name"

			for _, team := range testTeams {
				var teamBytes []byte

				teamBytes, err = getTeamBytes(team)
				So(err, ShouldBeNil)

				err = client.HSet(ctx, teamsKey, team.ID, teamBytes).Err()
				So(err, ShouldBeNil)
			}

			err = fillTeamNamesHash(logger, db)
			So(err, ShouldBeNil)

			var actualTeamNames map[string]string

			actualTeamNames, err = client.HGetAll(ctx, teamsByNamesKey).Result()
			So(err, ShouldBeNil)
			So(actualTeamNames, ShouldHaveLength, len(testTeams))

			expectedLowercasedTeamNames := []string{"team name", "team name1", "team name2", strings.ToLower(testTeams[3].Name)}
			for _, name := range expectedLowercasedTeamNames {
				_, ok := actualTeamNames[name]
				So(ok, ShouldBeTrue)
			}

			for i, team := range testTeams {
				Convey(fmt.Sprintf("for team %v fields ok", i), func() {
					var marshaledTeam string

					marshaledTeam, err = client.HGet(ctx, teamsKey, team.ID).Result()
					So(err, ShouldBeNil)

					var actualTeam teamWithID

					actualTeam, err = unmarshalTeam(team.ID, []byte(marshaledTeam))
					So(err, ShouldBeNil)
					So(actualTeam.ID, ShouldEqual, team.ID)
					So(actualTeam.Description, ShouldEqual, team.Description)

					if i < 3 {
						So(actualTeam.Name, ShouldBeIn, []string{team.Name, team.Name + "1", team.Name + "2"})
					} else {
						So(actualTeam.Name, ShouldEqual, team.Name)
					}
				})
			}
		})

		Convey("with teams has no unique names and adding one number does not help", func() {
			testTeams[0].Name = "Team name"
			testTeams[1].Name = "teaM name"
			testTeams[2].Name = "Team name1"

			for _, team := range testTeams {
				var teamBytes []byte

				teamBytes, err = getTeamBytes(team)
				So(err, ShouldBeNil)

				err = client.HSet(ctx, teamsKey, team.ID, teamBytes).Err()
				So(err, ShouldBeNil)
			}

			err = fillTeamNamesHash(logger, db)
			So(err, ShouldBeNil)

			var actualTeamNames map[string]string

			actualTeamNames, err = client.HGetAll(ctx, teamsByNamesKey).Result()
			So(err, ShouldBeNil)
			So(actualTeamNames, ShouldHaveLength, len(testTeams))

			// depends on order of map iteration
			expectedLowercasedTeamNames := []string{"team name", "team name1", "team name1_0", "team name1_1", strings.ToLower(testTeams[3].Name)}
			for name := range actualTeamNames {
				So(name, ShouldBeIn, expectedLowercasedTeamNames)
			}

			for i, team := range testTeams {
				Convey(fmt.Sprintf("for team %v fields ok", i), func() {
					var marshaledTeam string

					marshaledTeam, err = client.HGet(ctx, teamsKey, team.ID).Result()
					So(err, ShouldBeNil)

					var actualTeam teamWithID

					actualTeam, err = unmarshalTeam(team.ID, []byte(marshaledTeam))
					So(err, ShouldBeNil)
					So(actualTeam.ID, ShouldEqual, team.ID)
					So(actualTeam.Description, ShouldEqual, team.Description)

					if i < 3 {
						So(actualTeam.Name, ShouldBeIn, []string{team.Name, team.Name + "1", team.Name + "1_1", team.Name + "_0"})
					} else {
						So(actualTeam.Name, ShouldEqual, team.Name)
					}
				})
			}
		})
	})
}

func Test_removeTeamNamesHash(t *testing.T) {
	Convey("Test removing \"moira-teams-by-names\" hash", t, func() {
		conf := getDefault()

		logger, err := logging.ConfigureLog(conf.LogFile, conf.LogLevel, "test", conf.LogPrettyFormat)
		if err != nil {
			t.Fatal(err)
		}

		db := redis.NewTestDatabase(logger)

		db.Flush()
		defer db.Flush()

		ctx := context.Background()
		client := db.Client()

		Convey("with empty database", func() {
			err = removeTeamNamesHash(logger, db)
			So(err, ShouldBeNil)

			res, existErr := client.Exists(ctx, teamsByNamesKey).Result()
			So(existErr, ShouldBeNil)
			So(res, ShouldEqual, 0)
		})

		Convey("with filled teams and teams by names hashes", func() {
			defer db.Flush()

			for _, team := range testTeams {
				var teamBytes []byte

				teamBytes, err = getTeamBytes(team)
				So(err, ShouldBeNil)

				err = client.HSet(ctx, teamsKey, team.ID, teamBytes).Err()
				So(err, ShouldBeNil)

				err = client.HSet(ctx, teamsByNamesKey, strings.ToLower(team.Name), team.ID).Err()
				So(err, ShouldBeNil)
			}

			err = removeTeamNamesHash(logger, db)
			So(err, ShouldBeNil)

			res, existErr := client.Exists(ctx, teamsByNamesKey).Result()
			So(existErr, ShouldBeNil)
			So(res, ShouldEqual, 0)
		})
	})
}
