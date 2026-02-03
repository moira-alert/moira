package redis

import (
	"fmt"
	"strings"
	"testing"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/clock"
	"github.com/moira-alert/moira/database"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	. "github.com/smartystreets/goconvey/convey"
)

func TestTeamStoring(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database test in short mode")
	}

	logger, _ := logging.GetLogger("dataBase")
	dataBase := NewTestDatabase(logger, clock.NewSystemClock())
	dataBase.Flush()

	defer dataBase.Flush()

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

		actualTeam, err = dataBase.GetTeamByName(team.Name)
		So(err, ShouldBeNil)
		So(actualTeam, ShouldResemble, team)

		actualTeam, err = dataBase.GetTeam("nonExistentTeam")
		So(err, ShouldResemble, database.ErrNil)
		So(actualTeam, ShouldResemble, moira.Team{})

		// Add two users for team 1
		err = dataBase.SaveTeamsAndUsers(
			teamID,
			[]string{userID, userID2},
			map[string][]string{
				userID:  {teamID},
				userID2: {teamID},
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
		So(actualTeams, ShouldHaveLength, 1)
		So(actualTeams, ShouldContain, teamID)

		actualTeams, err = dataBase.GetUserTeams(userID2)
		So(err, ShouldBeNil)
		So(actualTeams, ShouldHaveLength, 1)
		So(actualTeams, ShouldContain, teamID)

		// Remove user 2 from team 1
		err = dataBase.SaveTeamsAndUsers(
			teamID,
			[]string{userID},
			map[string][]string{
				userID:  {teamID},
				userID2: {},
			},
		)
		So(err, ShouldBeNil)

		actualUsers, err = dataBase.GetTeamUsers(teamID)
		So(err, ShouldBeNil)
		So(len(actualUsers), ShouldResemble, 1)
		So(actualUsers, ShouldContain, userID)

		actualTeams, err = dataBase.GetUserTeams(userID)
		So(err, ShouldBeNil)
		So(actualTeams, ShouldHaveLength, 1)
		So(actualTeams, ShouldContain, teamID)

		actualTeams, err = dataBase.GetUserTeams(userID2)
		So(err, ShouldBeNil)
		So(actualTeams, ShouldHaveLength, 0)

		// Saving some users for team to check users existence in team later
		err = dataBase.SaveTeamsAndUsers(
			teamID2,
			[]string{userID, userID3},
			map[string][]string{
				userID:  {teamID, teamID},
				userID3: {teamID2},
			},
		)
		So(err, ShouldBeNil)

		actualUserExists, err := dataBase.IsTeamContainUser(teamID2, userID)
		So(err, ShouldBeNil)
		So(actualUserExists, ShouldBeTrue)

		actualUserExists, err = dataBase.IsTeamContainUser(teamID2, "NonexistentUser")
		So(err, ShouldBeNil)
		So(actualUserExists, ShouldBeFalse)

		actualUserExists, err = dataBase.IsTeamContainUser("NonexistentTeam", "NonexistentUser")
		So(err, ShouldBeNil)
		So(actualUserExists, ShouldBeFalse)

		actualTeams, err = dataBase.GetUserTeams(userID)
		So(err, ShouldBeNil)
		So(actualTeams, ShouldResemble, []string{teamID})

		actualTeams, err = dataBase.GetUserTeams("nonexistentUser")
		So(err, ShouldBeNil)
		So(actualTeams, ShouldResemble, []string{})

		// Add user to new team and delete this team
		const teamToDeleteID = "teamToDeleteID"

		const userOfTeamToDeleteID = "userOfTeamToDeleteID"

		teamToDelete := moira.Team{
			Name:        "TeamName",
			Description: "Team Description",
		}

		err = dataBase.SaveTeam(teamToDeleteID, teamToDelete)
		So(err, ShouldBeNil)

		err = dataBase.SaveTeamsAndUsers(teamToDeleteID, []string{userOfTeamToDeleteID}, map[string][]string{teamToDeleteID: {userOfTeamToDeleteID}})
		So(err, ShouldBeNil)

		err = dataBase.DeleteTeam(teamToDeleteID, userOfTeamToDeleteID)
		So(err, ShouldBeNil)

		actualTeam, err = dataBase.GetTeam(teamToDeleteID)
		So(err, ShouldResemble, database.ErrNil)
		So(actualTeam, ShouldResemble, moira.Team{})

		actualTeam, err = dataBase.GetTeamByName(teamToDelete.Name)
		So(err, ShouldResemble, database.ErrNil)
		So(actualTeam, ShouldResemble, moira.Team{})

		actualTeams, err = dataBase.GetUserTeams(userOfTeamToDeleteID)
		So(err, ShouldBeNil)
		So(actualTeams, ShouldHaveLength, 0)

		actualUsers, err = dataBase.GetTeamUsers(teamToDeleteID)
		So(err, ShouldBeNil)
		So(actualUsers, ShouldHaveLength, 0)
	})
}

func TestGetAllTeams(t *testing.T) {
	Convey("Test getting all teams", t, func() {
		logger, _ := logging.GetLogger("dataBase")
		dataBase := NewTestDatabase(logger, clock.NewSystemClock())
		dataBase.Flush()

		defer dataBase.Flush()

		Convey("with empty db returns no err and empty teams slice", func() {
			teams, err := dataBase.GetAllTeams()
			So(err, ShouldBeNil)
			So(teams, ShouldHaveLength, 0)
		})

		testTeams := []moira.Team{
			{
				ID:   "teamID_1",
				Name: "First team",
			},
			{
				ID:   "teamID_2",
				Name: "Second team",
			},
			{
				ID:   "teamID_3",
				Name: "Third team",
			},
			{
				ID:   "teamID_4",
				Name: "Fourth team",
			},
			{
				ID:   "teamID_5",
				Name: "Fifth team",
			},
		}

		Convey("with some teams returns all", func() {
			type expectedTeamCase struct {
				moira.Team

				count int
			}

			mapOfExpectedTeams := make(map[string]expectedTeamCase)

			for _, team := range testTeams {
				mapOfExpectedTeams[team.ID] = expectedTeamCase{
					Team:  team,
					count: 0,
				}

				err := dataBase.SaveTeam(team.ID, team)
				So(err, ShouldBeNil)
			}

			gotTeams, err := dataBase.GetAllTeams()
			So(err, ShouldBeNil)
			So(gotTeams, ShouldHaveLength, len(mapOfExpectedTeams))

			Convey("check equality of teams", func() {
				for _, team := range gotTeams {
					Convey(fmt.Sprintf("for team with id: %s", team.ID), func() {
						expectedTeam, exists := mapOfExpectedTeams[team.ID]
						So(exists, ShouldBeTrue)
						So(team, ShouldResemble, expectedTeam.Team)

						if exists {
							expectedTeam.count += 1
							mapOfExpectedTeams[team.ID] = expectedTeam
							So(expectedTeam.count, ShouldEqual, 1)
						}
					})
				}
			})
		})
	})
}

func TestSaveAndGetTeam(t *testing.T) {
	Convey("Test saving team", t, func() {
		logger, _ := logging.GetLogger("dataBase")
		dataBase := NewTestDatabase(logger, clock.NewSystemClock())
		dataBase.Flush()

		defer dataBase.Flush()

		team := moira.Team{
			ID:          "someTeamID",
			Name:        "Test team name",
			Description: "Test description",
		}

		Convey("when no team, get returns database.ErrNil", func() {
			gotTeam, err := dataBase.GetTeam(team.ID)

			So(err, ShouldResemble, database.ErrNil)
			So(gotTeam, ShouldResemble, moira.Team{})
		})

		Convey("when no team, get by name returns database.ErrNil", func() {
			gotTeam, err := dataBase.GetTeamByName(team.Name)

			So(err, ShouldResemble, database.ErrNil)
			So(gotTeam, ShouldResemble, moira.Team{})
		})

		Convey("when no team with such name, saved ok", func() {
			err := dataBase.SaveTeam(team.ID, team)

			So(err, ShouldBeNil)

			Convey("and getting team by id returns saved team", func() {
				gotTeam, err := dataBase.GetTeam(team.ID)

				So(err, ShouldBeNil)
				So(gotTeam, ShouldResemble, team)
			})

			Convey("and getting team by name returns saved team", func() {
				gotTeam, err := dataBase.GetTeamByName(team.Name)

				So(err, ShouldBeNil)
				So(gotTeam, ShouldResemble, team)
			})

			Convey("and updating team ok", func() {
				team.Name = strings.ToUpper(team.Name)

				err := dataBase.SaveTeam(team.ID, team)
				So(err, ShouldBeNil)

				gotTeam, err := dataBase.GetTeam(team.ID)
				So(err, ShouldBeNil)
				So(gotTeam, ShouldResemble, team)

				gotTeam, err = dataBase.GetTeamByName(team.Name)
				So(err, ShouldBeNil)
				So(gotTeam, ShouldResemble, team)
			})
		})

		Convey("with changing name of existing team", func() {
			err := dataBase.SaveTeam(team.ID, team)
			So(err, ShouldBeNil)

			otherTeam := moira.Team{
				ID:          "otherTeamID",
				Name:        "Other team name",
				Description: "others description",
			}

			err = dataBase.SaveTeam(otherTeam.ID, otherTeam)
			So(err, ShouldBeNil)

			prevName := otherTeam.Name

			Convey("to name of existed team (no matter upper/lower-case) returns error", func() {
				otherTeam.Name = team.Name

				err = dataBase.SaveTeam(otherTeam.ID, otherTeam)
				So(err, ShouldResemble, database.ErrTeamWithNameAlreadyExists)

				otherTeam.Name = strings.ToLower(team.Name)

				err = dataBase.SaveTeam(otherTeam.ID, otherTeam)
				So(err, ShouldResemble, database.ErrTeamWithNameAlreadyExists)
			})

			Convey("to new name, no team with prev name exist", func() {
				otherTeam.Name = team.Name + "1"

				err = dataBase.SaveTeam(otherTeam.ID, otherTeam)
				So(err, ShouldBeNil)

				gotTeam, err := dataBase.GetTeam(otherTeam.ID)
				So(err, ShouldBeNil)
				So(gotTeam, ShouldResemble, otherTeam)

				gotTeam, err = dataBase.GetTeamByName(prevName)
				So(err, ShouldResemble, database.ErrNil)
				So(gotTeam, ShouldResemble, moira.Team{})

				gotTeam, err = dataBase.GetTeamByName(otherTeam.Name)
				So(err, ShouldBeNil)
				So(gotTeam, ShouldResemble, otherTeam)
			})
		})
	})
}
