// nolint
package dto

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/middleware"
	mock "github.com/moira-alert/moira/mock/moira-alert"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/mock/gomock"
)

func TestSubscription_checkContacts(t *testing.T) {
	Convey("checkContacts", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		dataBase := mock.NewMockDatabase(mockCtrl)

		auth := &api.Authorization{Enabled: false}

		subscription := Subscription{}
		const userID = "userID"
		const teamID = "teamID"
		const contactID = "contactID"
		const contactID2 = "contactID2"
		responseWriter := httptest.NewRecorder()

		Convey("For user", func() {
			request := httptest.NewRequest(http.MethodPost, "/api/subscriptions", strings.NewReader(""))
			request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "auth", auth))
			middleware.DatabaseContext(dataBase)(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				request = req
			})).ServeHTTP(responseWriter, request)

			request.Header.Add("x-webauth-user", userID)
			middleware.UserContext(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				request = req
			})).ServeHTTP(responseWriter, request)

			Convey("All subscription contacts are for user", func() {
				subscription.Contacts = []string{contactID}
				dataBase.EXPECT().GetUserContactIDs(userID).Return([]string{contactID, contactID2}, nil)
				err := subscription.checkContacts(request)
				So(err, ShouldBeNil)
			})
			Convey("Subscription contact is another user contact", func() {
				subscription.Contacts = []string{contactID}
				dataBase.EXPECT().GetUserContactIDs(userID).Return([]string{contactID2}, nil)
				dataBase.EXPECT().GetContacts([]string{contactID}).Return([]*moira.ContactData{{ID: contactID, Value: "test value"}}, nil)
				err := subscription.checkContacts(request)
				So(err, ShouldResemble, ErrProvidedContactsForbidden{contactNames: []string{"test value"}, contactIds: []string{contactID}})
			})
		})

		Convey("For team", func() {
			request := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/teams/%s/subscriptions", teamID), strings.NewReader(""))
			request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "auth", auth))
			middleware.DatabaseContext(dataBase)(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				request = req
			})).ServeHTTP(responseWriter, request)
			request.Header.Add("x-webauth-user", userID)
			middleware.UserContext(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				request = req
			})).ServeHTTP(responseWriter, request)

			Convey("All subscription contacts are for current team", func() {
				subscription.Contacts = []string{contactID}
				request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "teamID", teamID))
				dataBase.EXPECT().GetTeamContactIDs(teamID).Return([]string{contactID, contactID2}, nil)
				err := subscription.checkContacts(request)
				So(err, ShouldBeNil)
			})
			Convey("All subscription contacts are for current team, but teamID placed in subscription JSON(subscription update case)", func() {
				subscription.TeamID = teamID
				subscription.Contacts = []string{contactID}
				dataBase.EXPECT().GetTeamContactIDs(teamID).Return([]string{contactID, contactID2}, nil)
				err := subscription.checkContacts(request)
				So(err, ShouldBeNil)
			})
		})

		Convey("Error bot teamID and userID specified in JSON", func() {
			request := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/teams/%s/subscriptions", teamID), strings.NewReader(""))
			request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "auth", auth))
			middleware.DatabaseContext(dataBase)(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				request = req
			})).ServeHTTP(responseWriter, request)
			request.Header.Add("x-webauth-user", userID)
			middleware.UserContext(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				request = req
			})).ServeHTTP(responseWriter, request)
			subscription.Contacts = []string{contactID}
			subscription.TeamID = teamID
			subscription.User = userID

			err := subscription.checkContacts(request)
			So(err, ShouldResemble, ErrSubscriptionContainsTeamAndUser{})
		})
	})
}
