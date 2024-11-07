package dto

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/middleware"
	"github.com/moira-alert/moira/datatypes"
	. "github.com/smartystreets/goconvey/convey"
)

var (
	testContactID = "test-contact-id"

	testEmergencyContact = datatypes.EmergencyContact{
		ContactID:      testContactID,
		HeartbeatTypes: []datatypes.HeartbeatType{datatypes.HeartbeatNotifier},
	}
)

func TestEmergencyContactBind(t *testing.T) {
	auth := &api.Authorization{
		Enabled: true,
		AllowedEmergencyContactTypes: map[datatypes.HeartbeatType]struct{}{
			datatypes.HeartbeatNotifier: {},
			datatypes.HeartbeatDatabase: {},
		},
	}

	userLogin := "test"
	testLoginKey := "login"
	testAuthKey := "auth"
	testContactID := "test-contact-id"

	Convey("Test Bind", t, func() {
		Convey("With empty emergency types", func() {
			testRequest := httptest.NewRequest(http.MethodPut, "/contact", http.NoBody)
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testLoginKey, userLogin))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testAuthKey, auth))

			emergencyContact := &EmergencyContact{
				ContactID: testContactID,
			}

			err := emergencyContact.Bind(testRequest)
			So(err, ShouldEqual, ErrEmptyHeartbeatTypes)
		})

		Convey("With invalid heartbeat type", func() {
			testRequest := httptest.NewRequest(http.MethodPut, "/contact", http.NoBody)
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testLoginKey, userLogin))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testAuthKey, auth))

			emergencyContact := &EmergencyContact{
				ContactID:      testContactID,
				HeartbeatTypes: []datatypes.HeartbeatType{"invalid-heartbeat-type"},
			}

			err := emergencyContact.Bind(testRequest)
			So(err, ShouldResemble, fmt.Errorf("'invalid-heartbeat-type' heartbeat type doesn't exist"))
		})

		Convey("With not allowed heartbeat type", func() {
			testRequest := httptest.NewRequest(http.MethodPut, "/contact", http.NoBody)
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testLoginKey, userLogin))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testAuthKey, auth))

			emergencyContact := &EmergencyContact{
				ContactID:      testContactID,
				HeartbeatTypes: []datatypes.HeartbeatType{datatypes.HeartbeatFilter},
			}

			err := emergencyContact.Bind(testRequest)
			So(err, ShouldResemble, fmt.Errorf("'%s' heartbeat type is not allowed", datatypes.HeartbeatFilter))
		})

		Convey("With allowed heartbeat types", func() {
			testRequest := httptest.NewRequest(http.MethodPut, "/contact", http.NoBody)
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testLoginKey, userLogin))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testAuthKey, auth))

			emergencyContact := &EmergencyContact{
				ContactID:      testContactID,
				HeartbeatTypes: []datatypes.HeartbeatType{datatypes.HeartbeatDatabase, datatypes.HeartbeatNotifier},
			}

			err := emergencyContact.Bind(testRequest)
			So(err, ShouldBeNil)
		})

		Convey("With admin who's allowed everything", func() {
			testRequest := httptest.NewRequest(http.MethodPut, "/contact", http.NoBody)
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testLoginKey, userLogin))
			testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), testAuthKey, auth))

			auth.AdminList = map[string]struct{}{
				userLogin: {},
			}

			emergencyContact := &EmergencyContact{
				ContactID:      testContactID,
				HeartbeatTypes: []datatypes.HeartbeatType{datatypes.HeartbeatFilter, datatypes.HeartbeatNotifier},
			}

			err := emergencyContact.Bind(testRequest)
			So(err, ShouldBeNil)
		})
	})
}

func TestFromEmergencyContacts(t *testing.T) {
	Convey("Test FromEmergencyContacts", t, func() {
		Convey("With nil emergency contacts", func() {
			expectedEmergencyContactList := &EmergencyContactList{
				List: make([]EmergencyContact, 0),
			}
			emergencyContactList := FromEmergencyContacts(nil)
			So(emergencyContactList, ShouldResemble, expectedEmergencyContactList)
		})

		Convey("With some emergency contacts", func() {
			expectedEmergencyContactList := &EmergencyContactList{
				List: []EmergencyContact{
					EmergencyContact(testEmergencyContact),
				},
			}
			emergencyContacts := []*datatypes.EmergencyContact{
				&testEmergencyContact,
				nil,
			}
			emergencyContactList := FromEmergencyContacts(emergencyContacts)
			So(emergencyContactList, ShouldResemble, expectedEmergencyContactList)
		})
	})
}
