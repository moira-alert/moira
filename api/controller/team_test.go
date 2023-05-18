package controller

import (
	"errors"
	"fmt"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/dto"
	"github.com/moira-alert/moira/database"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
)

func TestCreateTeam(t *testing.T) {
	Convey("CreateTeam", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)

		const user = "userID"
		team := dto.TeamModel{Name: "testTeam", Description: "test team description"}

		Convey("create successfully", func() {
			ID := ""
			dataBase.EXPECT().GetTeam(gomock.Any()).Return(moira.Team{}, database.ErrNil)
			dataBase.EXPECT().SaveTeam(gomock.Any(), team.ToMoiraTeam()).DoAndReturn(func(teamID string, moiraTeam moira.Team) error {
				ID = teamID
				return nil
			})
			dataBase.EXPECT().GetUserTeams(user).Return([]string{ID}, nil)
			dataBase.EXPECT().SaveTeamsAndUsers(gomock.Any(), []string{user}, gomock.Any()).Return(nil)
			response, err := CreateTeam(dataBase, team, user)
			So(response.ID, ShouldResemble, ID)
			So(err, ShouldBeNil)
		})

		Convey("create successfully with ID", func() {
			teamID := "teamID"
			team.ID = teamID
			dataBase.EXPECT().GetTeam(teamID).Return(moira.Team{}, database.ErrNil)
			dataBase.EXPECT().SaveTeam(teamID, team.ToMoiraTeam()).Return(nil)
			dataBase.EXPECT().GetUserTeams(user).Return([]string{}, nil)
			dataBase.EXPECT().SaveTeamsAndUsers(teamID, []string{user}, map[string][]string{user: {teamID}}).Return(nil)
			response, err := CreateTeam(dataBase, team, user)
			So(response.ID, ShouldResemble, teamID)
			So(err, ShouldBeNil)
		})

		Convey("team with this UUID exists", func() {
			ID := ""
			dataBase.EXPECT().GetTeam(gomock.Any()).Return(moira.Team{}, nil)
			dataBase.EXPECT().GetTeam(gomock.Any()).Return(moira.Team{}, database.ErrNil)
			dataBase.EXPECT().SaveTeam(gomock.Any(), team.ToMoiraTeam()).DoAndReturn(func(teamID string, moiraTeam moira.Team) error {
				ID = teamID
				return nil
			})
			dataBase.EXPECT().GetUserTeams(user).Return([]string{ID}, nil)
			dataBase.EXPECT().SaveTeamsAndUsers(gomock.Any(), []string{user}, gomock.Any()).Return(nil)
			response, err := CreateTeam(dataBase, team, user)
			So(response.ID, ShouldResemble, ID)
			So(err, ShouldBeNil)
		})

		Convey("team with this UUID exists and all retries failed", func() {
			dataBase.EXPECT().GetTeam(gomock.Any()).Return(moira.Team{}, nil)
			dataBase.EXPECT().GetTeam(gomock.Any()).Return(moira.Team{}, nil)
			dataBase.EXPECT().GetTeam(gomock.Any()).Return(moira.Team{}, nil)
			response, err := CreateTeam(dataBase, team, user)
			So(err, ShouldResemble, api.ErrorInternalServer(fmt.Errorf("cannot generate unique id for team")))
			So(response, ShouldResemble, dto.SaveTeamResponse{})
		})

		Convey("save error", func() {
			returnErr := fmt.Errorf("unexpected error")
			dataBase.EXPECT().GetTeam(gomock.Any()).Return(moira.Team{}, database.ErrNil)
			dataBase.EXPECT().SaveTeam(gomock.Any(), team.ToMoiraTeam()).Return(returnErr)
			response, err := CreateTeam(dataBase, team, user)
			So(response, ShouldResemble, dto.SaveTeamResponse{})
			So(err, ShouldResemble, api.ErrorInternalServer(fmt.Errorf("cannot save team: %w", returnErr)))
		})
	})
}

func TestDeleteTeam(t *testing.T) {
	Convey("DeleteTeam", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)

		const teamID = "temaID"
		const userID = "userID"
		errReturned := fmt.Errorf("test error")

		Convey("delete successfully", func() {
			gomock.InOrder(
				dataBase.EXPECT().GetTeamUsers(teamID).Return([]string{userID}, nil),
				dataBase.EXPECT().GetTeamContactIDs(teamID).Return([]string{}, nil),
				dataBase.EXPECT().GetTeamSubscriptionIDs(teamID).Return([]string{}, nil),
				dataBase.EXPECT().DeleteTeam(teamID, userID).Return(nil),
			)
			response, err := DeleteTeam(dataBase, teamID, userID)
			So(err, ShouldBeNil)
			So(response, ShouldResemble, dto.SaveTeamResponse{ID: teamID})
		})

		Convey("team have subscriptions", func() {
			gomock.InOrder(
				dataBase.EXPECT().GetTeamUsers(teamID).Return([]string{userID}, nil),
				dataBase.EXPECT().GetTeamContactIDs(teamID).Return([]string{}, nil),
				dataBase.EXPECT().GetTeamSubscriptionIDs(teamID).Return([]string{"subscriptionID"}, nil),
			)
			response, err := DeleteTeam(dataBase, teamID, userID)
			So(err, ShouldResemble, api.ErrorInvalidRequest(fmt.Errorf("cannot delete team: team have subscriptions: subscriptionID")))
			So(response, ShouldResemble, dto.SaveTeamResponse{})
		})
		Convey("error in get team subscriptions", func() {
			gomock.InOrder(
				dataBase.EXPECT().GetTeamUsers(teamID).Return([]string{userID}, nil),
				dataBase.EXPECT().GetTeamContactIDs(teamID).Return([]string{}, nil),
				dataBase.EXPECT().GetTeamSubscriptionIDs(teamID).Return([]string{}, errReturned),
			)
			response, err := DeleteTeam(dataBase, teamID, userID)
			So(err, ShouldResemble, api.ErrorInternalServer(fmt.Errorf("cannot get team subscriptions: %w", errReturned)))
			So(response, ShouldResemble, dto.SaveTeamResponse{})
		})
		Convey("team have contacts", func() {
			gomock.InOrder(
				dataBase.EXPECT().GetTeamUsers(teamID).Return([]string{userID}, nil),
				dataBase.EXPECT().GetTeamContactIDs(teamID).Return([]string{"contactID"}, nil),
			)
			response, err := DeleteTeam(dataBase, teamID, userID)
			So(err, ShouldResemble, api.ErrorInvalidRequest(fmt.Errorf("cannot delete team: team have contacts: contactID")))
			So(response, ShouldResemble, dto.SaveTeamResponse{})
		})
		Convey("error in get team contacts", func() {
			gomock.InOrder(
				dataBase.EXPECT().GetTeamUsers(teamID).Return([]string{userID}, nil),
				dataBase.EXPECT().GetTeamContactIDs(teamID).Return([]string{}, errReturned),
			)
			response, err := DeleteTeam(dataBase, teamID, userID)
			So(err, ShouldResemble, api.ErrorInternalServer(fmt.Errorf("cannot get team contacts: %w", errReturned)))
			So(response, ShouldResemble, dto.SaveTeamResponse{})
		})
		Convey("team have more than one user", func() {
			dataBase.EXPECT().GetTeamUsers(teamID).Return([]string{userID, "anotherUserID"}, nil)
			response, err := DeleteTeam(dataBase, teamID, userID)
			So(err, ShouldResemble, api.ErrorInvalidRequest(fmt.Errorf("cannot delete team: team have users: userID, anotherUserID")))
			So(response, ShouldResemble, dto.SaveTeamResponse{})
		})
		Convey("error in get team users", func() {
			dataBase.EXPECT().GetTeamUsers(teamID).Return([]string{}, errReturned)
			response, err := DeleteTeam(dataBase, teamID, userID)
			So(err, ShouldResemble, api.ErrorInternalServer(fmt.Errorf("cannot get team users: %w", errReturned)))
			So(response, ShouldResemble, dto.SaveTeamResponse{})
		})
	})
}

func TestGetTeam(t *testing.T) {
	Convey("GetTeam", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)

		const teamID = "testTeam"
		team := moira.Team{Name: "testTeam", Description: "test team description"}

		Convey("get successfully", func() {
			dataBase.EXPECT().GetTeam(teamID).Return(team, nil)
			response, err := GetTeam(dataBase, teamID)
			So(response, ShouldResemble, dto.NewTeamModel(team))
			So(err, ShouldBeNil)
		})

		Convey("team not found", func() {
			dataBase.EXPECT().GetTeam(teamID).Return(moira.Team{}, database.ErrNil)
			response, err := GetTeam(dataBase, teamID)
			So(response, ShouldResemble, dto.TeamModel{})
			So(err, ShouldResemble, api.ErrorNotFound("cannot find team: testTeam"))
		})

		Convey("database error", func() {
			returnErr := fmt.Errorf("unexpected error")
			dataBase.EXPECT().GetTeam(teamID).Return(moira.Team{}, returnErr)
			response, err := GetTeam(dataBase, teamID)
			So(response, ShouldResemble, dto.TeamModel{})
			So(err, ShouldResemble, api.ErrorInternalServer(fmt.Errorf("cannot get team from database: %w", returnErr)))
		})
	})
}

func TestGetUserTeams(t *testing.T) {
	Convey("GetUserTeams", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)

		const userID = "userID"
		const teamID = "team1"
		const teamID2 = "team2"
		teamsIDs := []string{teamID, teamID2}
		teams := []dto.TeamModel{
			{
				ID:          teamID,
				Name:        "team 1 name",
				Description: "team 1 Description",
			},
			{
				ID:          teamID2,
				Name:        "team 2 name",
				Description: "team 2 Description",
			},
		}

		Convey("get successfully", func() {
			dataBase.EXPECT().GetUserTeams(userID).Return(teamsIDs, nil)
			dataBase.EXPECT().GetTeam(teamID).Return(teams[0].ToMoiraTeam(), nil)
			dataBase.EXPECT().GetTeam(teamID2).Return(teams[1].ToMoiraTeam(), nil)
			response, err := GetUserTeams(dataBase, userID)
			So(response, ShouldResemble, dto.UserTeams{Teams: teams})
			So(err, ShouldBeNil)
		})

		Convey("teams not found", func() {
			dataBase.EXPECT().GetUserTeams(userID).Return([]string{}, database.ErrNil)
			response, err := GetUserTeams(dataBase, userID)
			So(response, ShouldResemble, dto.UserTeams{Teams: []dto.TeamModel{}})
			So(err, ShouldBeNil)
		})

		Convey("database error", func() {
			returnErr := fmt.Errorf("unexpected error")
			dataBase.EXPECT().GetUserTeams(userID).Return([]string{}, returnErr)
			response, err := GetUserTeams(dataBase, userID)
			So(response, ShouldResemble, dto.UserTeams{})
			So(err, ShouldResemble, api.ErrorInternalServer(fmt.Errorf("cannot get user teams from database: %w", returnErr)))
		})
	})
}

func TestGetTeamUsers(t *testing.T) {
	Convey("GetTeamUsers", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)

		const teamID = "testTeam"
		users := []string{"userID1", "userID2"}

		Convey("get successfully", func() {
			dataBase.EXPECT().GetTeamUsers(teamID).Return(users, nil)
			response, err := GetTeamUsers(dataBase, teamID)
			So(response, ShouldResemble, dto.TeamMembers{Usernames: users})
			So(err, ShouldBeNil)
		})

		Convey("users not found", func() {
			dataBase.EXPECT().GetTeamUsers(teamID).Return([]string{}, database.ErrNil)
			response, err := GetTeamUsers(dataBase, teamID)
			So(response, ShouldResemble, dto.TeamMembers{})
			So(err, ShouldResemble, api.ErrorNotFound("cannot find team users: testTeam"))
		})

		Convey("database error", func() {
			returnErr := fmt.Errorf("unexpected error")
			dataBase.EXPECT().GetTeamUsers(teamID).Return([]string{}, returnErr)
			response, err := GetTeamUsers(dataBase, teamID)
			So(response, ShouldResemble, dto.TeamMembers{})
			So(err, ShouldResemble, api.ErrorInternalServer(fmt.Errorf("cannot get team users from database: %w", returnErr)))
		})
	})
}

func TestAddTeamUsers(t *testing.T) {
	Convey("AddTeamUsers", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)

		const teamID = "testTeam"
		const userID = "userID"
		const userID2 = "userID2"
		const userID3 = "userID3"

		Convey("add successfully", func() {
			gomock.InOrder(
				dataBase.EXPECT().GetTeamUsers(teamID).Return([]string{userID, userID2}, nil),
				dataBase.EXPECT().GetUserTeams(userID).Return([]string{teamID}, nil),
				dataBase.EXPECT().GetUserTeams(userID2).Return([]string{teamID}, nil),
				dataBase.EXPECT().GetUserTeams(userID3).Return([]string{}, nil),
				dataBase.EXPECT().SaveTeamsAndUsers(teamID,
					[]string{userID, userID2, userID3},
					map[string][]string{
						userID:  {teamID},
						userID2: {teamID},
						userID3: {teamID},
					},
				).Return(nil),
			)
			response, err := AddTeamUsers(dataBase, teamID, []string{userID3})
			So(response, ShouldResemble, dto.TeamMembers{Usernames: []string{userID, userID2, userID3}})
			So(err, ShouldBeNil)
		})

		Convey("team users not found", func() {
			dataBase.EXPECT().GetTeamUsers(teamID).Return([]string{}, database.ErrNil)
			response, err := AddTeamUsers(dataBase, teamID, []string{userID3})
			So(response, ShouldResemble, dto.TeamMembers{})
			So(err, ShouldResemble, api.ErrorNotFound("cannot find team users: testTeam"))
		})

		Convey("user teams not found", func() {
			gomock.InOrder(
				dataBase.EXPECT().GetTeamUsers(teamID).Return([]string{userID, userID2}, nil),
				dataBase.EXPECT().GetUserTeams(userID).Return([]string{}, database.ErrNil),
			)
			response, err := AddTeamUsers(dataBase, teamID, []string{userID3})
			So(response, ShouldResemble, dto.TeamMembers{})
			So(err, ShouldResemble, api.ErrorNotFound("cannot find user teams: userID"))
		})

		Convey("user already exists", func() {
			gomock.InOrder(
				dataBase.EXPECT().GetTeamUsers(teamID).Return([]string{userID, userID2, userID3}, nil),
				dataBase.EXPECT().GetUserTeams(userID).Return([]string{teamID}, nil),
				dataBase.EXPECT().GetUserTeams(userID2).Return([]string{teamID}, nil),
				dataBase.EXPECT().GetUserTeams(userID3).Return([]string{teamID}, nil),
			)
			response, err := AddTeamUsers(dataBase, teamID, []string{userID3})
			So(response, ShouldResemble, dto.TeamMembers{})
			So(err, ShouldResemble, api.ErrorInvalidRequest(fmt.Errorf("one ore many users you specified are already exist in this team: userID3")))
		})
	})
}

func Test_addUserTeam(t *testing.T) {
	Convey("addUserTeam", t, func() {
		Convey("add successfully", func() {
			actual, err := addUserTeam("testTeam3", []string{"testTeam", "testTeam2"})
			So(actual, ShouldResemble, []string{"testTeam", "testTeam2", "testTeam3"})
			So(err, ShouldBeNil)
		})

		Convey("team already exists", func() {
			actual, err := addUserTeam("testTeam", []string{"testTeam", "testTeam2"})
			So(actual, ShouldResemble, []string{})
			So(err, ShouldResemble, fmt.Errorf("team already exist in user teams: testTeam"))
		})
	})
}

func TestUpdateTeam(t *testing.T) {
	Convey("UpdateTeam", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)

		const teamID = "testTeam"
		team := dto.TeamModel{Name: "testTeam", Description: "test team description"}

		Convey("update successfully", func() {
			dataBase.EXPECT().SaveTeam(teamID, team.ToMoiraTeam()).Return(nil)
			response, err := UpdateTeam(dataBase, teamID, team)
			So(response.ID, ShouldResemble, teamID)
			So(err, ShouldBeNil)
		})
	})
}

func TestDeleteTeamUser(t *testing.T) {
	Convey("DeleteTeamUser", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)

		const teamID = "testTeam"
		const userID = "userID"
		const userID2 = "userID2"
		const userID3 = "userID3"

		Convey("user exists", func() {
			gomock.InOrder(
				dataBase.EXPECT().GetTeamUsers(teamID).Return([]string{userID, userID2, userID3}, nil),
				dataBase.EXPECT().GetUserTeams(userID).Return([]string{teamID, "team2"}, nil),
				dataBase.EXPECT().GetUserTeams(userID2).Return([]string{teamID}, nil),
				dataBase.EXPECT().GetUserTeams(userID3).Return([]string{teamID}, nil),
				dataBase.EXPECT().SaveTeamsAndUsers(teamID, []string{userID2, userID3}, map[string][]string{
					userID:  {"team2"},
					userID2: {teamID},
					userID3: {teamID},
				}).Return(nil),
			)
			reply, err := DeleteTeamUser(dataBase, teamID, userID)
			So(reply, ShouldResemble, dto.TeamMembers{Usernames: []string{userID2, userID3}})
			So(err, ShouldBeNil)
		})
		Convey("team does not have any users", func() {
			dataBase.EXPECT().GetTeamUsers(teamID).Return([]string{}, database.ErrNil)
			reply, err := DeleteTeamUser(dataBase, teamID, userID)
			So(reply, ShouldResemble, dto.TeamMembers{})
			So(err, ShouldResemble, api.ErrorNotFound("cannot find team users: testTeam"))
		})
		Convey("removal of last user", func() {
			dataBase.EXPECT().GetTeamUsers(teamID).Return([]string{userID}, nil)
			reply, err := DeleteTeamUser(dataBase, teamID, userID)
			So(reply, ShouldResemble, dto.TeamMembers{})
			So(err, ShouldResemble, api.ErrorInvalidRequest(fmt.Errorf("cannot remove last member of team")))
		})
		Convey("team does not contain user", func() {
			dataBase.EXPECT().GetTeamUsers(teamID).Return([]string{userID2, userID3}, nil)
			reply, err := DeleteTeamUser(dataBase, teamID, userID)
			So(reply, ShouldResemble, dto.TeamMembers{})
			So(err, ShouldResemble, api.ErrorNotFound("user that you specified not found in this team: userID"))
		})
		Convey("one user do not have teams", func() {
			gomock.InOrder(
				dataBase.EXPECT().GetTeamUsers(teamID).Return([]string{userID, userID2, userID3}, nil),
				dataBase.EXPECT().GetUserTeams(userID).Return([]string{}, database.ErrNil),
			)
			reply, err := DeleteTeamUser(dataBase, teamID, userID)
			So(reply, ShouldResemble, dto.TeamMembers{})
			So(err, ShouldResemble, api.ErrorNotFound("cannot find user teams: userID"))
		})
	})
}

func Test_removeUserTeam(t *testing.T) {
	Convey("removeUserTeam", t, func() {
		Convey("remove successfully", func() {
			actual, err := removeUserTeam([]string{"testTeam", "testTeam2"}, "testTeam")
			So(actual, ShouldResemble, []string{"testTeam2"})
			So(err, ShouldBeNil)
		})

		Convey("team not found", func() {
			actual, err := removeUserTeam([]string{"testTeam1", "testTeam2"}, "testTeam")
			So(actual, ShouldResemble, []string{})
			So(err, ShouldResemble, fmt.Errorf("cannot find team in user teams: testTeam"))
		})
	})
}

func Test_fillCurrentUsersTeamsMap(t *testing.T) {
	Convey("fillCurrentUsersTeamsMap", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)

		const teamID = "testTeam"
		const userID1 = "userID1"
		const userID2 = "userID2"

		Convey("without error", func() {
			dataBase.EXPECT().GetUserTeams(userID1).Return([]string{teamID, "team2"}, nil)
			dataBase.EXPECT().GetUserTeams(userID2).Return([]string{teamID}, nil)
			usersMap, err := fillCurrentUsersTeamsMap(dataBase, []string{userID1, userID2})
			So(err, ShouldBeNil)
			So(usersMap, ShouldHaveLength, 2)
			So(usersMap, ShouldContainKey, userID1)
			So(usersMap, ShouldContainKey, userID2)
			So(usersMap[userID1], ShouldHaveLength, 2)
			So(usersMap[userID2], ShouldHaveLength, 1)
		})
		Convey("with error", func() {
			errorReturned := errors.New("empty error")
			dataBase.EXPECT().GetUserTeams(userID1).Return([]string{}, errorReturned)
			usersMap, err := fillCurrentUsersTeamsMap(dataBase, []string{userID1, userID2})
			So(err, ShouldResemble, api.ErrorInternalServer(fmt.Errorf("cannot get team users from database: %w", errorReturned)))
			So(usersMap, ShouldHaveLength, 0)
		})
	})
}

func Test_removeDeletedUsers(t *testing.T) {
	Convey("removeDeletedUsers", t, func() {
		const teamID = "testTeam"
		const userID1 = "userID1"
		const userID2 = "userID2"
		const userID3 = "userID3"
		Convey("remove successful", func() {
			actual, err := removeDeletedUsers(
				teamID,
				[]string{userID1, userID2},
				map[string]bool{userID1: true, userID3: true},
				map[string][]string{
					userID1: {teamID, "otherTeam"},
					userID2: {teamID, "otherTeam"},
				})
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, map[string][]string{
				userID1: {teamID, "otherTeam"},
				userID2: {"otherTeam"},
			})
		})
		Convey("with error", func() {
			actual, err := removeDeletedUsers(
				teamID,
				[]string{userID1, userID2},
				map[string]bool{userID1: true, userID3: true},
				map[string][]string{
					userID1: {teamID, "otherTeam"},
					userID2: {"otherTeam"},
				})
			So(err, ShouldResemble, api.ErrorInternalServer(fmt.Errorf("cannot remove team from user: %w", fmt.Errorf("cannot find team in user teams: %s", teamID))))
			So(actual, ShouldBeNil)
		})
	})
}

func Test_addTeamsForNewUsers(t *testing.T) {
	Convey("addTeamsForNewUsers", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)

		const teamID = "testTeam"
		const userID1 = "userID1"
		const userID2 = "userID2"

		Convey("without error", func() {
			dataBase.EXPECT().GetUserTeams(userID2).Return([]string{"otherTeam2"}, nil)
			actual, err := addTeamsForNewUsers(
				dataBase,
				teamID,
				map[string]bool{
					userID1: true,
					userID2: true,
				},
				map[string][]string{
					userID1: {teamID, "otherTeam"},
				})
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, map[string][]string{
				userID1: {teamID, "otherTeam"},
				userID2: {"otherTeam2", teamID},
			})
		})
		Convey("with db error", func() {
			errReturned := errors.New("test")
			dataBase.EXPECT().GetUserTeams(userID2).Return(nil, errReturned)
			actual, err := addTeamsForNewUsers(
				dataBase,
				teamID,
				map[string]bool{
					userID1: true,
					userID2: true,
				},
				map[string][]string{
					userID1: {teamID, "otherTeam"},
				})
			So(err, ShouldResemble, api.ErrorInternalServer(fmt.Errorf("cannot get team users from database: %w", errReturned)))
			So(actual, ShouldBeNil)
		})
	})
}

func TestSetTeamUsers(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)

	const teamID = "testTeam"
	const userID1 = "userID1"
	const userID2 = "userID2"

	Convey("SetTeamUsers", t, func() {
		Convey("Set to empty team", func() {
			dataBase.EXPECT().GetTeamUsers(teamID).Return([]string{}, nil)
			dataBase.EXPECT().GetUserTeams(userID1).Return(nil, database.ErrNil)
			dataBase.EXPECT().SaveTeamsAndUsers(teamID, []string{userID1}, map[string][]string{userID1: {teamID}})
			actual, err := SetTeamUsers(dataBase, teamID, []string{userID1})
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, dto.TeamMembers{Usernames: []string{userID1}})
		})
		Convey("Set to team with members", func() {
			dataBase.EXPECT().GetTeamUsers(teamID).Return([]string{userID1}, nil)
			dataBase.EXPECT().GetUserTeams(userID1).Return([]string{teamID}, nil)
			dataBase.EXPECT().GetUserTeams(userID2).Return(nil, database.ErrNil)
			dataBase.EXPECT().SaveTeamsAndUsers(teamID, []string{userID1, userID2}, map[string][]string{userID1: {teamID}, userID2: {teamID}})
			actual, err := SetTeamUsers(dataBase, teamID, []string{userID1, userID2})
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, dto.TeamMembers{Usernames: []string{userID1, userID2}})
		})
	})
}

func TestCheckUserPermissionsForTeam(t *testing.T) {
	const teamID = "testTeam"
	const userID = "userID"

	Convey("CheckUserPermissionsForTeam", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)

		Convey("user in team", func() {
			dataBase.EXPECT().GetTeam(teamID).Return(moira.Team{}, nil)
			dataBase.EXPECT().IsTeamContainUser(teamID, userID).Return(true, nil)
			err := CheckUserPermissionsForTeam(dataBase, teamID, userID)
			So(err, ShouldBeNil)
		})
		Convey("user is not in team", func() {
			dataBase.EXPECT().GetTeam(teamID).Return(moira.Team{}, nil)
			dataBase.EXPECT().IsTeamContainUser(teamID, userID).Return(false, nil)
			err := CheckUserPermissionsForTeam(dataBase, teamID, userID)
			So(err, ShouldResemble, api.ErrorForbidden("you are not permitted to manipulate with this team"))
		})
		Convey("error while checking user", func() {
			returnErr := errors.New("returning error")
			dataBase.EXPECT().GetTeam(teamID).Return(moira.Team{}, nil)
			dataBase.EXPECT().IsTeamContainUser(teamID, userID).Return(false, returnErr)
			err := CheckUserPermissionsForTeam(dataBase, teamID, userID)
			So(err, ShouldResemble, api.ErrorInternalServer(returnErr))
		})
		Convey("error while getting team", func() {
			returnErr := errors.New("returning error")
			dataBase.EXPECT().GetTeam(teamID).Return(moira.Team{}, returnErr)
			err := CheckUserPermissionsForTeam(dataBase, teamID, userID)
			So(err, ShouldResemble, api.ErrorInternalServer(returnErr))
		})
		Convey("team is not exist", func() {
			dataBase.EXPECT().GetTeam(teamID).Return(moira.Team{}, database.ErrNil)
			err := CheckUserPermissionsForTeam(dataBase, teamID, userID)
			So(err, ShouldResemble, api.ErrorNotFound("team with ID 'testTeam' does not exists"))
		})
	})
}

func TestGetTeamSettings(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	teamID := "testTeam"

	Convey("Success get team settings", t, func() {
		subscriptionIDs := []string{uuid.Must(uuid.NewV4()).String(), uuid.Must(uuid.NewV4()).String()}
		subscriptions := []*moira.SubscriptionData{{ID: subscriptionIDs[0]}, {ID: subscriptionIDs[1]}}
		contactIDs := []string{uuid.Must(uuid.NewV4()).String(), uuid.Must(uuid.NewV4()).String()}
		contacts := []*moira.ContactData{{ID: contactIDs[0]}, {ID: contactIDs[1]}}
		database.EXPECT().GetTeamSubscriptionIDs(teamID).Return(subscriptionIDs, nil)
		database.EXPECT().GetSubscriptions(subscriptionIDs).Return(subscriptions, nil)
		database.EXPECT().GetTeamContactIDs(teamID).Return(contactIDs, nil)
		database.EXPECT().GetContacts(contactIDs).Return(contacts, nil)
		settings, err := GetTeamSettings(database, teamID)
		So(err, ShouldBeNil)
		So(settings, ShouldResemble, dto.TeamSettings{
			TeamID:        teamID,
			Contacts:      []moira.ContactData{*contacts[0], *contacts[1]},
			Subscriptions: []moira.SubscriptionData{*subscriptions[0], *subscriptions[1]},
		})
	})

	Convey("No contacts and subscriptions", t, func() {
		database.EXPECT().GetTeamSubscriptionIDs(teamID).Return(make([]string, 0), nil)
		database.EXPECT().GetSubscriptions(make([]string, 0)).Return(make([]*moira.SubscriptionData, 0), nil)
		database.EXPECT().GetTeamContactIDs(teamID).Return(make([]string, 0), nil)
		database.EXPECT().GetContacts(make([]string, 0)).Return(make([]*moira.ContactData, 0), nil)
		settings, err := GetTeamSettings(database, teamID)
		So(err, ShouldBeNil)
		So(settings, ShouldResemble, dto.TeamSettings{
			TeamID:        teamID,
			Contacts:      make([]moira.ContactData, 0),
			Subscriptions: make([]moira.SubscriptionData, 0),
		})
	})

	Convey("Errors", t, func() {
		Convey("GetTeamSubscriptionIDs", func() {
			expected := fmt.Errorf("can not read ids")
			database.EXPECT().GetTeamSubscriptionIDs(teamID).Return(nil, expected)
			settings, err := GetTeamSettings(database, teamID)
			So(err, ShouldResemble, api.ErrorInternalServer(expected))
			So(settings, ShouldResemble, dto.TeamSettings{})
		})
		Convey("GetSubscriptions", func() {
			expected := fmt.Errorf("can not read subscriptions")
			database.EXPECT().GetTeamSubscriptionIDs(teamID).Return(make([]string, 0), nil)
			database.EXPECT().GetSubscriptions(make([]string, 0)).Return(nil, expected)
			settings, err := GetTeamSettings(database, teamID)
			So(err, ShouldResemble, api.ErrorInternalServer(expected))
			So(settings, ShouldResemble, dto.TeamSettings{})
		})
		Convey("GetTeamContactIDs", func() {
			expected := fmt.Errorf("can not read contact ids")
			database.EXPECT().GetTeamSubscriptionIDs(teamID).Return(make([]string, 0), nil)
			database.EXPECT().GetSubscriptions(make([]string, 0)).Return(make([]*moira.SubscriptionData, 0), nil)
			database.EXPECT().GetTeamContactIDs(teamID).Return(nil, expected)
			settings, err := GetTeamSettings(database, teamID)
			So(err, ShouldResemble, api.ErrorInternalServer(expected))
			So(settings, ShouldResemble, dto.TeamSettings{})
		})
		Convey("GetContacts", func() {
			expected := fmt.Errorf("can not read contacts")
			subscriptionIDs := []string{uuid.Must(uuid.NewV4()).String(), uuid.Must(uuid.NewV4()).String()}
			subscriptions := []*moira.SubscriptionData{{ID: subscriptionIDs[0]}, {ID: subscriptionIDs[1]}}
			contactIDs := []string{uuid.Must(uuid.NewV4()).String(), uuid.Must(uuid.NewV4()).String()}
			database.EXPECT().GetTeamSubscriptionIDs(teamID).Return(subscriptionIDs, nil)
			database.EXPECT().GetSubscriptions(subscriptionIDs).Return(subscriptions, nil)
			database.EXPECT().GetTeamContactIDs(teamID).Return(contactIDs, nil)
			database.EXPECT().GetContacts(contactIDs).Return(nil, expected)
			settings, err := GetTeamSettings(database, teamID)
			So(err, ShouldResemble, api.ErrorInternalServer(expected))
			So(settings, ShouldResemble, dto.TeamSettings{})
		})
	})
}

func TestGetTeamSubsStats(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)

	const (
		team1ID   = "1"
		team1Name = "team1Name"

		team2ID   = "2"
		team2Name = "team2Name"
	)
	teams := []*moira.Team{
		{
			ID:   team1ID,
			Name: team1Name,
		},
		{
			ID:   team2ID,
			Name: team2Name,
		},
	}
	const (
		team1Contact1ID = "team1_contact1"
		team1Contact2ID = "team1_contact2"
		team1Contact3ID = "team1_contact3"
		contactType1    = "contact-type1"
		contactType2    = "contact-type2"
	)
	team1Contacts := []*moira.ContactData{
		{
			ID:   team1Contact1ID,
			Type: contactType1,
		},
		{
			ID:   team1Contact2ID,
			Type: contactType2,
		},
		{
			ID:   team1Contact3ID,
			Type: contactType1,
		},
	}
	team2Contacts := []*moira.ContactData{
		{
			Type: contactType1,
		},
		{
			Type: contactType1,
		},
	}

	const (
		tag1 = "tag1"
		tag2 = "tag2"
		tag3 = "tag3"
	)
	const (
		team1Sub1 = "team1_sub1"
		team1Sub2 = "team1_sub2"
		team1Sub3 = "team1_sub3"

		team2Sub1 = "team2_sub1"
	)
	team1Subs := []*moira.SubscriptionData{
		{
			ID:       team1Sub1,
			Enabled:  true,
			Contacts: []string{team1Contact1ID, team1Contact2ID},
			TeamID:   team1ID,
			Tags:     []string{tag1},
		},
		{
			ID:       team1Sub2,
			Enabled:  true,
			Contacts: []string{team1Contact1ID, team1Contact2ID},
			TeamID:   team1ID,
			Tags:     []string{tag1, tag2},
		},
		{
			ID:       team1Sub3,
			Enabled:  true,
			Contacts: []string{team1Contact1ID, team1Contact2ID},
			TeamID:   team1ID,
			Tags:     []string{tag1, tag2, tag3},
		},
	}
	const abandonedTag = "abandoned"
	const team2ContactID = "team2_contact"
	team2Subs := []*moira.SubscriptionData{
		{
			ID:       team2Sub1,
			Enabled:  false,
			Contacts: []string{team2ContactID},
			TeamID:   team2ID,
			Tags:     []string{abandonedTag},
		},
	}

	Convey("GetTeamSubsStats should return statistics", t, func() {
		var nilErr error = nil
		dataBase.EXPECT().GetAllTeams().Return(teams, nilErr)

		// team1
		dataBase.EXPECT().GetTeamSubscriptionIDs(team1ID).Return([]string{team1Sub1, team1Sub2, team1Sub3}, nilErr)
		dataBase.EXPECT().GetTeamContactIDs(team1ID).Return([]string{team1Contact1ID, team1Contact2ID, team1Contact3ID}, nilErr)
		dataBase.EXPECT().GetTeamUsers(team1ID).Return([]string{"team1_user1", "team1_user2"}, nilErr)
		dataBase.EXPECT().GetContacts([]string{team1Contact1ID, team1Contact2ID, team1Contact3ID}).Return(team1Contacts, nilErr)
		dataBase.EXPECT().GetSubscriptions([]string{team1Sub1, team1Sub2, team1Sub3}).Return(team1Subs, nilErr)

		// team2
		dataBase.EXPECT().GetTeamSubscriptionIDs(team2ID).Return([]string{team2Sub1}, nilErr)
		dataBase.EXPECT().GetTeamContactIDs(team2ID).Return([]string{"team2_contact1", "team2_contact2"}, nilErr)
		dataBase.EXPECT().GetTeamUsers(team2ID).Return([]string{"team2_user1"}, nilErr)
		dataBase.EXPECT().GetContacts([]string{"team2_contact1", "team2_contact2"}).Return(team2Contacts, nilErr)
		dataBase.EXPECT().GetSubscriptions([]string{team2Sub1}).Return(team2Subs, nilErr)

		stats, err := GetTeamSubsStats(dataBase, logger)
		So(err, ShouldBeNil)
		So(stats, ShouldHaveLength, 2)

		statsExpected := dto.TeamSubsStats{
			{
				TeamID:             team1ID,
				TeamName:           teams[0].Name,
				SubscriptionsCount: 3,
				ContactsCount:      3,
				UsersCount:         2,
				UniqueSendersCount: 2,
				UniqueTagsCount:    3,
			},
			{
				TeamID:             team2ID,
				TeamName:           teams[1].Name,
				SubscriptionsCount: 1,
				ContactsCount:      2,
				UsersCount:         1,
				UniqueSendersCount: 1,
				UniqueTagsCount:    1,
			},
		}
		assert.ElementsMatch(t, stats, statsExpected)
	})
}

//func TestCollectStatistic(t *testing.T) {
//	logger, _ := logging.GetLogger("dataBase")
//	dataBase := redis.NewTestDatabase(logger)
//
//	stats, err := GetTeamSubsStats(dataBase, logger)
//	assert.Nil(t, err)
//
//	file, _ := os.Create("./stats-wo-triggers-29-05.csv")
//	defer file.Close()
//
//	var data [][]string
//	data = append(data, []string{
//		"TeamID",
//		"TeamName",
//		"SubscriptionsCount",
//		"UsersCount",
//		"ContactsCount",
//		"UniqueSendersCount",
//		"UniqueTagsCount",
//	})
//	for _, stat := range stats {
//		row := []string{
//			stat.TeamID,
//			stat.TeamName,
//			strconv.Itoa(stat.SubscriptionsCount),
//			strconv.Itoa(stat.UsersCount),
//			strconv.Itoa(stat.ContactsCount),
//			strconv.Itoa(stat.UniqueSendersCount),
//			strconv.Itoa(stat.UniqueTagsCount),
//		}
//		data = append(data, row)
//	}
//	w := csv.NewWriter(file)
//	err = w.WriteAll(data) // calls Flush internally
//	assert.Nil(t, err)
//}
//

func TestGetTeamTriggersStats(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)

	const (
		team1ID   = "1"
		team1Name = "team1Name"

		team2ID   = "2"
		team2Name = "team2Name"

		team3ID   = "3"
		team3Name = "team3Name"
	)
	teams := []*moira.Team{
		{
			ID:   team1ID,
			Name: team1Name,
		},
		{
			ID:   team2ID,
			Name: team2Name,
		},
		{
			ID:   team3ID,
			Name: team3Name,
		},
	}

	const (
		tag1 = "tag1"
		tag2 = "tag2"
		tag3 = "tag3"
	)
	const (
		sub1 = "sub1"
		sub2 = "sub2"
		sub3 = "sub3"
	)
	teamSubs := []*moira.SubscriptionData{
		{
			ID:      sub1,
			Enabled: true,
			TeamID:  team1ID,
			Tags:    []string{tag1},
		},
		{
			ID:      sub2,
			Enabled: true,
			TeamID:  team2ID,
			Tags:    []string{tag1, tag2},
		},
		{
			ID:      sub3,
			Enabled: true,
			TeamID:  team3ID,
			Tags:    []string{tag1, tag2, tag3},
		},
	}

	const (
		trigger1ID = "trigger1"
		trigger2ID = "trigger2"
		trigger3ID = "trigger3"
	)

	triggers := []*moira.Trigger{
		{
			ID:   trigger1ID,
			Tags: []string{tag1},
		},
		{
			ID:   trigger2ID,
			Tags: []string{tag1, tag2},
		},
		{
			ID:   trigger3ID,
			Tags: []string{tag1, tag2, tag3},
		},
	}

	Convey("GetTeamTriggersStats should return statistics", t, func() {
		var nilErr error = nil
		dataBase.EXPECT().GetTagNames().Return([]string{tag1, tag2, tag3}, nilErr).AnyTimes()

		dataBase.EXPECT().GetTagsSubscriptions([]string{tag1}).Return([]*moira.SubscriptionData{teamSubs[0], teamSubs[1], teamSubs[2]}, nilErr)
		dataBase.EXPECT().GetTagsSubscriptions([]string{tag2}).Return([]*moira.SubscriptionData{teamSubs[1], teamSubs[2]}, nilErr)
		dataBase.EXPECT().GetTagsSubscriptions([]string{tag3}).Return([]*moira.SubscriptionData{teamSubs[2]}, nilErr)

		dataBase.EXPECT().GetTagTriggerIDs(tag1).Return([]string{trigger1ID, trigger2ID, trigger3ID}, nilErr).AnyTimes()
		dataBase.EXPECT().GetTagTriggerIDs(tag2).Return([]string{trigger2ID, trigger3ID}, nilErr).AnyTimes()
		dataBase.EXPECT().GetTagTriggerIDs(tag3).Return([]string{trigger3ID}, nilErr).AnyTimes()

		dataBase.EXPECT().GetTrigger(trigger1ID).Return(*triggers[0], nilErr).AnyTimes()
		dataBase.EXPECT().GetTrigger(trigger2ID).Return(*triggers[1], nilErr).AnyTimes()
		dataBase.EXPECT().GetTrigger(trigger3ID).Return(*triggers[2], nilErr).AnyTimes()

		dataBase.EXPECT().GetAllTeams().Return(teams, nilErr)

		stats, err := GetTeamTriggersStats(dataBase, logger)
		So(err, ShouldBeNil)

		statsExpected := dto.TeamTriggersStats{
			{
				TeamID:        team1ID,
				TeamName:      team1Name,
				TriggersCount: 3,
			},
			{
				TeamID:        team2ID,
				TeamName:      team2Name,
				TriggersCount: 2,
			},
			{
				TeamID:        team3ID,
				TeamName:      team3Name,
				TriggersCount: 1,
			},
		}
		assert.ElementsMatch(t, stats, statsExpected)
	})
}

//func TestGetTeamTriggersStats_File(t *testing.T) {
//	logger, _ := logging.GetLogger("dataBase")
//	dataBase := redis.NewTestDatabase(logger)
//
//	stats, err := GetTeamTriggersStats(dataBase, logger)
//	logger.Info().Msg("done")
//	assert.Nil(t, err)
//
//	file, _ := os.Create("./team-triggers-29-05.csv")
//	defer file.Close()
//
//	var data [][]string
//	data = append(data, []string{
//		"TeamID",
//		"TeamName",
//		"TriggersCount",
//	})
//	for _, stat := range stats {
//		row := []string{
//			stat.TeamID,
//			stat.TeamName,
//			strconv.Itoa(stat.TriggersCount),
//		}
//		data = append(data, row)
//	}
//	w := csv.NewWriter(file)
//	errWrite := w.WriteAll(data) // calls Flush internally
//	assert.Nil(t, errWrite)
//}
