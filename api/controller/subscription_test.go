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
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	. "github.com/smartystreets/goconvey/convey"
)

func TestGetUserSubscriptions(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	const login = "user"

	Convey("Two subscriptions", t, func() {
		subscriptionIDs := []string{uuid.Must(uuid.NewV4()).String(), uuid.Must(uuid.NewV4()).String()}
		subscriptions := []*moira.SubscriptionData{{ID: subscriptionIDs[0]}, {ID: subscriptionIDs[1]}}
		database.EXPECT().GetUserSubscriptionIDs(login).Return(subscriptionIDs, nil)
		database.EXPECT().GetSubscriptions(subscriptionIDs).Return(subscriptions, nil)
		list, err := GetUserSubscriptions(database, login)
		So(err, ShouldBeNil)
		So(list, ShouldResemble, &dto.SubscriptionList{List: []moira.SubscriptionData{*subscriptions[0], *subscriptions[1]}})
	})

	Convey("Two ids, one subscription", t, func() {
		subscriptionIDs := []string{uuid.Must(uuid.NewV4()).String(), uuid.Must(uuid.NewV4()).String()}
		subscriptions := []*moira.SubscriptionData{{ID: subscriptionIDs[1]}}
		database.EXPECT().GetUserSubscriptionIDs(login).Return(subscriptionIDs, nil)
		database.EXPECT().GetSubscriptions(subscriptionIDs).Return(subscriptions, nil)
		list, err := GetUserSubscriptions(database, login)
		So(err, ShouldBeNil)
		So(list, ShouldResemble, &dto.SubscriptionList{List: []moira.SubscriptionData{*subscriptions[0]}})
	})

	Convey("Errors", t, func() {
		Convey("GetUserSubscriptionIDs", func() {
			expected := fmt.Errorf("oh no!!!11 Cant get subscription ids")
			database.EXPECT().GetUserSubscriptionIDs(login).Return(nil, expected)
			list, err := GetUserSubscriptions(database, login)
			So(err, ShouldResemble, api.ErrorInternalServer(expected))
			So(list, ShouldBeNil)
		})

		Convey("GetSubscriptions", func() {
			expected := fmt.Errorf("oh no!!!11 Cant get subscriptions")
			subscriptionIDs := []string{uuid.Must(uuid.NewV4()).String(), uuid.Must(uuid.NewV4()).String()}
			database.EXPECT().GetUserSubscriptionIDs(login).Return(subscriptionIDs, nil)
			database.EXPECT().GetSubscriptions(subscriptionIDs).Return(nil, expected)
			list, err := GetUserSubscriptions(database, login)
			So(err, ShouldResemble, api.ErrorInternalServer(expected))
			So(list, ShouldBeNil)
		})
	})
}

func TestUpdateSubscription(t *testing.T) {
	Convey("UpdateSubscription", t, func() {
		mockCtrl := gomock.NewController(t)
		dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
		defer mockCtrl.Finish()
		userLogin := "user"
		teamID := "team"

		Convey("Success update for user", func() {
			subscriptionDTO := &dto.Subscription{}
			subscriptionID := uuid.Must(uuid.NewV4()).String()
			subscription := moira.SubscriptionData{
				ID:   subscriptionID,
				User: userLogin,
			}
			dataBase.EXPECT().SaveSubscription(&subscription).Return(nil)
			err := UpdateSubscription(dataBase, subscriptionID, userLogin, subscriptionDTO)
			So(err, ShouldBeNil)
			So(subscriptionDTO.User, ShouldResemble, userLogin)
			So(subscriptionDTO.ID, ShouldResemble, subscriptionID)
		})

		Convey("Error save for user", func() {
			subscriptionDTO := &dto.Subscription{}
			subscriptionID := uuid.Must(uuid.NewV4()).String()
			subscription := moira.SubscriptionData{
				ID:   subscriptionID,
				User: userLogin,
			}
			err := fmt.Errorf("oooops")
			dataBase.EXPECT().SaveSubscription(&subscription).Return(err)
			actual := UpdateSubscription(dataBase, subscriptionID, userLogin, subscriptionDTO)
			So(actual, ShouldResemble, api.ErrorInternalServer(err))
			So(subscriptionDTO.User, ShouldResemble, userLogin)
			So(subscriptionDTO.ID, ShouldResemble, subscriptionID)
		})

		Convey("Success update for team", func() {
			subscriptionDTO := &dto.Subscription{TeamID: teamID}
			subscriptionID := uuid.Must(uuid.NewV4()).String()
			subscription := moira.SubscriptionData{
				ID:     subscriptionID,
				TeamID: teamID,
			}
			dataBase.EXPECT().SaveSubscription(&subscription).Return(nil)
			err := UpdateSubscription(dataBase, subscriptionID, userLogin, subscriptionDTO)
			So(err, ShouldBeNil)
			So(subscriptionDTO.TeamID, ShouldResemble, teamID)
			So(subscriptionDTO.ID, ShouldResemble, subscriptionID)
		})

		Convey("Error save for team", func() {
			subscriptionDTO := &dto.Subscription{TeamID: teamID}
			subscriptionID := uuid.Must(uuid.NewV4()).String()
			subscription := moira.SubscriptionData{
				ID:     subscriptionID,
				TeamID: teamID,
			}
			err := fmt.Errorf("oooops")
			dataBase.EXPECT().SaveSubscription(&subscription).Return(err)
			actual := UpdateSubscription(dataBase, subscriptionID, userLogin, subscriptionDTO)
			So(actual, ShouldResemble, api.ErrorInternalServer(err))
			So(subscriptionDTO.TeamID, ShouldResemble, teamID)
			So(subscriptionDTO.ID, ShouldResemble, subscriptionID)
		})
	})
}

func TestRemoveSubscription(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	db := mock_moira_alert.NewMockDatabase(mockCtrl)
	id := uuid.Must(uuid.NewV4()).String()

	Convey("Success", t, func() {
		db.EXPECT().RemoveSubscription(id).Return(nil)
		err := RemoveSubscription(db, id)
		So(err, ShouldBeNil)
	})

	Convey("Error", t, func() {
		expected := fmt.Errorf("oooops! Can not remove subscription")
		db.EXPECT().RemoveSubscription(id).Return(expected)
		err := RemoveSubscription(db, id)
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
	})
}

func TestSendTestNotification(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	db := mock_moira_alert.NewMockDatabase(mockCtrl)
	id := uuid.Must(uuid.NewV4()).String()

	Convey("Success", t, func() {
		db.EXPECT().PushNotificationEvent(gomock.Any(), false).Return(nil)
		err := SendTestNotification(db, id)
		So(err, ShouldBeNil)
	})

	Convey("Error", t, func() {
		expected := fmt.Errorf("oooops! Can not push event")
		db.EXPECT().PushNotificationEvent(gomock.Any(), false).Return(expected)
		err := SendTestNotification(db, id)
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
	})
}

func TestCreateSubscription(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	const login = "user"
	const teamID = "testTeam"
	auth := &api.Authorization{Enabled: false}

	Convey("Create for user", t, func() {
		Convey("Success create", func() {
			subscription := dto.Subscription{ID: ""}
			dataBase.EXPECT().SaveSubscription(gomock.Any()).Return(nil)
			err := CreateSubscription(dataBase, auth, login, "", &subscription)
			So(err, ShouldBeNil)
		})

		Convey("Success create subscription with id", func() {
			sub := &dto.Subscription{
				ID: uuid.Must(uuid.NewV4()).String(),
			}
			dataBase.EXPECT().GetSubscription(sub.ID).Return(moira.SubscriptionData{}, database.ErrNil)
			dataBase.EXPECT().SaveSubscription(gomock.Any()).Return(nil)
			err := CreateSubscription(dataBase, auth, login, "", sub)
			So(err, ShouldBeNil)
			So(sub.User, ShouldResemble, login)
			So(sub.ID, ShouldResemble, sub.ID)
		})

		Convey("Subscription exists by id", func() {
			subscription := &dto.Subscription{
				ID: uuid.Must(uuid.NewV4()).String(),
			}
			dataBase.EXPECT().GetSubscription(subscription.ID).Return(moira.SubscriptionData{}, nil)
			err := CreateSubscription(dataBase, auth, login, "", subscription)
			So(err, ShouldResemble, api.ErrorInvalidRequest(fmt.Errorf("subscription with this ID already exists")))
		})

		Convey("Error get subscription", func() {
			subscription := &dto.Subscription{
				ID: uuid.Must(uuid.NewV4()).String(),
			}
			err := fmt.Errorf("oooops! Can not write contact")
			dataBase.EXPECT().GetSubscription(subscription.ID).Return(moira.SubscriptionData{}, err)
			expected := CreateSubscription(dataBase, auth, login, "", subscription)
			So(expected, ShouldResemble, api.ErrorInternalServer(err))
		})

		Convey("Error save subscription", func() {
			subscription := dto.Subscription{ID: ""}
			expected := fmt.Errorf("oooops! Can not create subscription")
			dataBase.EXPECT().SaveSubscription(gomock.Any()).Return(expected)
			err := CreateSubscription(dataBase, auth, login, "", &subscription)
			So(err, ShouldResemble, api.ErrorInternalServer(expected))
		})
	})
	Convey("Create for team", t, func() {
		Convey("Success create", func() {
			subscription := dto.Subscription{ID: ""}
			dataBase.EXPECT().SaveSubscription(gomock.Any()).Return(nil)
			err := CreateSubscription(dataBase, auth, "", teamID, &subscription)
			So(err, ShouldBeNil)
		})

		Convey("Success create subscription with id", func() {
			sub := &dto.Subscription{
				ID: uuid.Must(uuid.NewV4()).String(),
			}
			dataBase.EXPECT().GetSubscription(sub.ID).Return(moira.SubscriptionData{}, database.ErrNil)
			dataBase.EXPECT().SaveSubscription(gomock.Any()).Return(nil)
			err := CreateSubscription(dataBase, auth, "", teamID, sub)
			So(err, ShouldBeNil)
			So(sub.TeamID, ShouldResemble, teamID)
			So(sub.ID, ShouldResemble, sub.ID)
		})

		Convey("Subscription exists by id", func() {
			subscription := &dto.Subscription{
				ID: uuid.Must(uuid.NewV4()).String(),
			}
			dataBase.EXPECT().GetSubscription(subscription.ID).Return(moira.SubscriptionData{}, nil)
			err := CreateSubscription(dataBase, auth, "", teamID, subscription)
			So(err, ShouldResemble, api.ErrorInvalidRequest(fmt.Errorf("subscription with this ID already exists")))
		})

		Convey("Error get subscription", func() {
			subscription := &dto.Subscription{
				ID: uuid.Must(uuid.NewV4()).String(),
			}
			err := fmt.Errorf("oooops! Can not write contact")
			dataBase.EXPECT().GetSubscription(subscription.ID).Return(moira.SubscriptionData{}, err)
			expected := CreateSubscription(dataBase, auth, "", teamID, subscription)
			So(expected, ShouldResemble, api.ErrorInternalServer(err))
		})

		Convey("Error save subscription", func() {
			subscription := dto.Subscription{ID: ""}
			expected := fmt.Errorf("oooops! Can not create subscription")
			dataBase.EXPECT().SaveSubscription(gomock.Any()).Return(expected)
			err := CreateSubscription(dataBase, auth, "", teamID, &subscription)
			So(err, ShouldResemble, api.ErrorInternalServer(expected))
		})
	})
	Convey("Error on create with both: userLogin and teamID specified", t, func() {
		subscription := &dto.Subscription{}
		err := CreateSubscription(dataBase, auth, login, teamID, subscription)
		So(err, ShouldResemble, api.ErrorInternalServer(fmt.Errorf("CreateSubscription: cannot create subscription when both userLogin and teamID specified")))
	})
}

func TestCheckUserPermissionsForSubscription(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	userLogin := uuid.Must(uuid.NewV4()).String()
	teamID := uuid.Must(uuid.NewV4()).String()
	id := uuid.Must(uuid.NewV4()).String()
	auth := &api.Authorization{}

	Convey("No subscription", t, func() {
		dataBase.EXPECT().GetSubscription(id).Return(moira.SubscriptionData{}, database.ErrNil)
		expectedSub, expected := CheckUserPermissionsForSubscription(dataBase, id, userLogin, auth)
		So(expected, ShouldResemble, api.ErrorNotFound(fmt.Sprintf("subscription with ID '%s' does not exists", id)))
		So(expectedSub, ShouldResemble, moira.SubscriptionData{})
	})

	Convey("Different user", t, func() {
		actualSub := moira.SubscriptionData{User: "diffUser"}
		dataBase.EXPECT().GetSubscription(id).Return(actualSub, nil)
		expectedSub, expected := CheckUserPermissionsForSubscription(dataBase, id, userLogin, auth)
		So(expected, ShouldResemble, api.ErrorForbidden("you are not permitted"))
		So(expectedSub, ShouldResemble, moira.SubscriptionData{})
	})

	Convey("Has subscription", t, func() {
		actualSub := moira.SubscriptionData{ID: id, User: userLogin}
		dataBase.EXPECT().GetSubscription(id).Return(actualSub, nil)
		expectedSub, expected := CheckUserPermissionsForSubscription(dataBase, id, userLogin, auth)
		So(expected, ShouldBeNil)
		So(expectedSub, ShouldResemble, actualSub)
	})

	Convey("Error get contact", t, func() {
		err := fmt.Errorf("oooops! Can not read contact")
		dataBase.EXPECT().GetSubscription(id).Return(moira.SubscriptionData{}, err)
		expectedSub, expected := CheckUserPermissionsForSubscription(dataBase, id, userLogin, auth)
		So(expected, ShouldResemble, api.ErrorInternalServer(err))
		So(expectedSub, ShouldResemble, moira.SubscriptionData{})
	})

	Convey("Team subscription", t, func() {
		Convey("User is in team", func() {
			expectedSub := moira.SubscriptionData{ID: id, TeamID: teamID}
			dataBase.EXPECT().GetSubscription(id).Return(expectedSub, nil)
			dataBase.EXPECT().IsTeamContainUser(teamID, userLogin).Return(true, nil)
			actual, err := CheckUserPermissionsForSubscription(dataBase, id, userLogin, auth)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, expectedSub)
		})
		Convey("User is not in team", func() {
			dataBase.EXPECT().GetSubscription(id).Return(moira.SubscriptionData{ID: id, TeamID: teamID}, nil)
			dataBase.EXPECT().IsTeamContainUser(teamID, userLogin).Return(false, nil)
			actual, err := CheckUserPermissionsForSubscription(dataBase, id, userLogin, auth)
			So(err, ShouldResemble, api.ErrorForbidden("you are not permitted"))
			So(actual, ShouldResemble, moira.SubscriptionData{})
		})
		Convey("Error while checking user team", func() {
			errReturned := errors.New("test error")
			dataBase.EXPECT().GetSubscription(id).Return(moira.SubscriptionData{ID: id, TeamID: teamID}, nil)
			dataBase.EXPECT().IsTeamContainUser(teamID, userLogin).Return(false, errReturned)
			actual, err := CheckUserPermissionsForSubscription(dataBase, id, userLogin, auth)
			So(err, ShouldResemble, api.ErrorInternalServer(errReturned))
			So(actual, ShouldResemble, moira.SubscriptionData{})
		})
	})
}

func TestCheckAdminPermissionsForSubscription(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	teamID := uuid.Must(uuid.NewV4()).String()
	id := uuid.Must(uuid.NewV4()).String()
	adminLogin := "admin_login"
	auth := &api.Authorization{Enabled: true, AdminList: map[string]struct{}{adminLogin: {}}}

	Convey("Same user", t, func() {
		expectedSub := moira.SubscriptionData{ID: id, User: adminLogin}
		dataBase.EXPECT().GetSubscription(id).Return(expectedSub, nil)
		actualContact, errorResponse := CheckUserPermissionsForSubscription(dataBase, id, adminLogin, auth)
		So(errorResponse, ShouldBeNil)
		So(actualContact, ShouldResemble, expectedSub)
	})

	Convey("Different user", t, func() {
		expectedSub := moira.SubscriptionData{ID: id, User: "diffUser"}
		dataBase.EXPECT().GetSubscription(id).Return(expectedSub, nil)
		actualContact, errorResponse := CheckUserPermissionsForSubscription(dataBase, id, adminLogin, auth)
		So(errorResponse, ShouldBeNil)
		So(actualContact, ShouldResemble, expectedSub)
	})

	Convey("Team contact", t, func() {
		expectedSub := moira.SubscriptionData{ID: id, TeamID: teamID}
		dataBase.EXPECT().GetSubscription(id).Return(expectedSub, nil)
		actualContact, errorResponse := CheckUserPermissionsForSubscription(dataBase, id, adminLogin, auth)
		So(errorResponse, ShouldBeNil)
		So(actualContact, ShouldResemble, expectedSub)
	})
}

func Test_isSubscriptionExists(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)

	subscriptionID := "testSubscription"
	subscription := moira.SubscriptionData{ID: subscriptionID}

	Convey("isSubscriptionExists", t, func() {
		Convey("subscription exists", func() {
			dataBase.EXPECT().GetSubscription(subscriptionID).Return(subscription, nil)
			actual, err := isSubscriptionExists(dataBase, subscriptionID)
			So(actual, ShouldBeTrue)
			So(err, ShouldBeNil)
		})
		Convey("subscription is not exist", func() {
			dataBase.EXPECT().GetSubscription(subscriptionID).Return(moira.SubscriptionData{}, database.ErrNil)
			actual, err := isSubscriptionExists(dataBase, subscriptionID)
			So(actual, ShouldBeFalse)
			So(err, ShouldBeNil)
		})
		Convey("error returned", func() {
			errReturned := errors.New("some error")
			dataBase.EXPECT().GetSubscription(subscriptionID).Return(moira.SubscriptionData{}, errReturned)
			actual, err := isSubscriptionExists(dataBase, subscriptionID)
			So(actual, ShouldBeFalse)
			So(err, ShouldResemble, errReturned)
		})
	})
}
