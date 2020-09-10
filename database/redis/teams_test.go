package redis

import (
	"testing"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	. "github.com/smartystreets/goconvey/convey"
)

func TestTeamStoring(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database test in short mode")
	}
	logger, _ := logging.GetLogger("dataBase")
	dataBase := newTestDatabase(logger, config)
	dataBase.flush()
	defer dataBase.flush()

	teamID := "testTeam"
	teamID2 := "testTeam2"
	userID := "userID"
	userID2 := "userID2"
	userID3 := "userID3"
	team := moira.Team{
		ID:          teamID,
		Name:        "Test team",
		Description: "Test team description",
	}
	Convey("Teams Manipulation", t, func() {
		err := dataBase.SaveTeam(teamID, team)
		So(err, ShouldBeNil)

		actualTeam, err := dataBase.GetTeam(teamID)
		So(err, ShouldBeNil)
		So(actualTeam, ShouldResemble, team)

		actualTeam, err = dataBase.GetTeam("nonExistentTeam")
		So(err, ShouldResemble, database.ErrNil)
		So(actualTeam, ShouldResemble, moira.Team{})

		err = dataBase.SaveTeamsAndUsers(
			teamID,
			[]string{userID, userID2},
			map[string][]string{
				userID:  {teamID},
				userID2: {teamID},
			},
		)
		So(err, ShouldBeNil)

		err = dataBase.SaveTeamsAndUsers(
			teamID2,
			[]string{userID, userID2},
			map[string][]string{
				userID:  {teamID, teamID},
				userID3: {teamID2},
			},
		)
		So(err, ShouldBeNil)

		actualUsers, err := dataBase.GetTeamUsers(teamID)
		So(err, ShouldBeNil)
		So(len(actualUsers), ShouldResemble, 2)
		So(actualUsers, ShouldContain, userID)
		So(actualUsers, ShouldContain, userID2)

		actualTeams, err := dataBase.GetUserTeams(userID)
		So(err, ShouldBeNil)
		So(actualTeams, ShouldResemble, []string{teamID})

		actualTeams, err = dataBase.GetUserTeams("nonexistentUser")
		So(err, ShouldBeNil)
		So(actualTeams, ShouldResemble, []string{})
	})
}
