package controller

import (
	"errors"
	"fmt"
	"testing"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/dto"
	moiradb "github.com/moira-alert/moira/database"
	"github.com/moira-alert/moira/datatypes"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/mock/gomock"
)

var (
	testContactID  = "test-contact-id"
	testContactID2 = "test-contact-id2"

	testEmergencyContact = datatypes.EmergencyContact{
		ContactID:      testContactID,
		HeartbeatTypes: []datatypes.HeartbeatType{datatypes.HeartbeatNotifier},
	}
	testEmergencyContact2 = datatypes.EmergencyContact{
		ContactID:      testContactID2,
		HeartbeatTypes: []datatypes.HeartbeatType{datatypes.HeartbeatTypeNotSet},
	}

	testEmergencyContactDTO = dto.EmergencyContact{
		ContactID:      testContactID,
		HeartbeatTypes: []datatypes.HeartbeatType{datatypes.HeartbeatNotifier},
	}
	testEmergencyContact2DTO = dto.EmergencyContact{
		ContactID:      testContactID2,
		HeartbeatTypes: []datatypes.HeartbeatType{datatypes.HeartbeatTypeNotSet},
	}
)

func TestGetEmergencyContacts(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	defer mockCtrl.Finish()

	Convey("Test GetEmergencyContacts", t, func() {
		Convey("With nil response from database", func() {
			database.EXPECT().GetEmergencyContacts().Return(nil, nil)
			expectedEmergencyContactList := &dto.EmergencyContactList{
				List: make([]dto.EmergencyContact, 0),
			}

			emergencyContactList, err := GetEmergencyContacts(database)
			So(err, ShouldBeNil)
			So(emergencyContactList, ShouldResemble, expectedEmergencyContactList)
		})

		Convey("With some saved emergency contacts in database", func() {
			database.EXPECT().GetEmergencyContacts().Return([]*datatypes.EmergencyContact{&testEmergencyContact, &testEmergencyContact2}, nil)
			expectedEmergencyContactList := &dto.EmergencyContactList{
				List: []dto.EmergencyContact{testEmergencyContactDTO, testEmergencyContact2DTO},
			}

			emergencyContactList, err := GetEmergencyContacts(database)
			So(err, ShouldBeNil)
			So(emergencyContactList, ShouldResemble, expectedEmergencyContactList)
		})
	})
}

func TestGetEmergencyContact(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	defer mockCtrl.Finish()

	Convey("Test GetEmergencyContact", t, func() {
		Convey("With unexisted emergency contact", func() {
			database.EXPECT().GetEmergencyContact(testContactID).Return(datatypes.EmergencyContact{}, moiradb.ErrNil)

			emergencyContact, err := GetEmergencyContact(database, testContactID)
			So(err, ShouldResemble, api.ErrorNotFound(fmt.Sprintf("emergency contact with ID '%s' does not exists", testContactID)))
			So(emergencyContact, ShouldBeNil)
		})

		Convey("With undefined db error", func() {
			expectedErr := errors.New("test-error")
			database.EXPECT().GetEmergencyContact(testContactID).Return(datatypes.EmergencyContact{}, expectedErr)

			emergencyContact, err := GetEmergencyContact(database, testContactID)
			So(err, ShouldResemble, api.ErrorInternalServer(expectedErr))
			So(emergencyContact, ShouldBeNil)
		})

		Convey("Successfully get emergency contact", func() {
			database.EXPECT().GetEmergencyContact(testContactID).Return(testEmergencyContact, nil)

			emergencyContact, err := GetEmergencyContact(database, testContactID)
			So(err, ShouldBeNil)
			So(emergencyContact, ShouldResemble, &testEmergencyContactDTO)
		})
	})
}

func TestVerifyEmergencyContactAccess(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	defer mockCtrl.Finish()

	const (
		admin = "admin"
		user  = "user"
	)

	contact := moira.ContactData{
		ID:   testContactID,
		User: user,
	}

	Convey("Test verifyEmergencyContactAccess", t, func() {
		Convey("With disabled auth and admin", func() {
			auth := &api.Authorization{
				AdminList: map[string]struct{}{
					admin: {},
				},
				Enabled: false,
			}

			database.EXPECT().GetContact(testContactID).Return(contact, nil)

			err := verifyEmergencyContactAccess(database, auth, testEmergencyContact, admin)
			So(err, ShouldResemble, api.ErrorInvalidRequest(fmt.Errorf("cannot create an emergency contact using someone else's contact_id '%s'", testContactID)))
		})

		Convey("With disabled auth and user", func() {
			auth := &api.Authorization{
				AdminList: map[string]struct{}{
					admin: {},
				},
				Enabled: false,
			}

			database.EXPECT().GetContact(testContactID).Return(contact, nil)

			err := verifyEmergencyContactAccess(database, auth, testEmergencyContact, user)
			So(err, ShouldBeNil)
		})

		Convey("With enabled auth and admin", func() {
			auth := &api.Authorization{
				AdminList: map[string]struct{}{
					admin: {},
				},
				Enabled: true,
			}

			database.EXPECT().GetContact(testContactID).Return(contact, nil)

			err := verifyEmergencyContactAccess(database, auth, testEmergencyContact, admin)
			So(err, ShouldBeNil)
		})

		Convey("With enabled auth and user", func() {
			auth := &api.Authorization{
				AdminList: map[string]struct{}{
					admin: {},
				},
				Enabled: true,
			}

			database.EXPECT().GetContact(testContactID).Return(contact, nil)

			err := verifyEmergencyContactAccess(database, auth, testEmergencyContact, user)
			So(err, ShouldBeNil)
		})

		Convey("With database error", func() {
			auth := &api.Authorization{
				AdminList: map[string]struct{}{
					admin: {},
				},
				Enabled: true,
			}

			dbErr := errors.New("get contact error")
			database.EXPECT().GetContact(testContactID).Return(moira.ContactData{}, dbErr)

			err := verifyEmergencyContactAccess(database, auth, testEmergencyContact, user)
			So(err, ShouldResemble, api.ErrorInternalServer(dbErr))
		})

		Convey("With empty userLogin", func() {
			auth := &api.Authorization{
				AdminList: map[string]struct{}{
					admin: {},
				},
				Enabled: true,
			}

			database.EXPECT().GetContact(testContactID).Return(contact, nil)

			err := verifyEmergencyContactAccess(database, auth, testEmergencyContact, user)
			So(err, ShouldBeNil)
		})
	})
}

func TestCreateEmergencyContact(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	defer mockCtrl.Finish()

	const (
		admin = "admin"
		user  = "user"
	)

	contact := moira.ContactData{
		ID:   testContactID,
		User: user,
	}

	auth := &api.Authorization{
		AdminList: map[string]struct{}{
			admin: {},
		},
		Enabled: true,
	}

	Convey("Test CreateEmergencyContact", t, func() {
		Convey("With nil emergency contact dto", func() {
			response, err := CreateEmergencyContact(database, auth, nil, user)
			So(err, ShouldBeNil)
			So(response, ShouldResemble, dto.SaveEmergencyContactResponse{})
		})

		Convey("With empty emergency contact id", func() {
			emergencyContactDTO := dto.EmergencyContact{}
			response, err := CreateEmergencyContact(database, auth, &emergencyContactDTO, user)
			So(err, ShouldResemble, api.ErrorInvalidRequest(ErrEmptyEmergencyContactID))
			So(response, ShouldResemble, dto.SaveEmergencyContactResponse{})
		})

		Convey("With get contact database error", func() {
			dbErr := errors.New("get contact error")
			database.EXPECT().GetContact(testContactID).Return(moira.ContactData{}, dbErr)

			response, err := CreateEmergencyContact(database, auth, &testEmergencyContactDTO, user)
			So(err, ShouldResemble, api.ErrorInternalServer(dbErr))
			So(response, ShouldResemble, dto.SaveEmergencyContactResponse{})
		})

		Convey("With save emergency contact database error", func() {
			dbErr := errors.New("create emergency contact error")
			database.EXPECT().GetContact(testContactID).Return(contact, nil)
			database.EXPECT().SaveEmergencyContact(testEmergencyContact).Return(dbErr)

			response, err := CreateEmergencyContact(database, auth, &testEmergencyContactDTO, user)
			So(err, ShouldResemble, api.ErrorInternalServer(dbErr))
			So(response, ShouldResemble, dto.SaveEmergencyContactResponse{})
		})

		Convey("Without any errors", func() {
			database.EXPECT().GetContact(testContactID).Return(contact, nil)
			database.EXPECT().SaveEmergencyContact(testEmergencyContact).Return(nil)

			response, err := CreateEmergencyContact(database, auth, &testEmergencyContactDTO, user)
			So(err, ShouldBeNil)
			So(response, ShouldResemble, dto.SaveEmergencyContactResponse{
				ContactID: testContactID,
			})
		})
	})
}

func TestUpdateEmergencyContact(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	defer mockCtrl.Finish()

	Convey("Test UpdateEmergencyContact", t, func() {
		Convey("With nil emergency contact dto", func() {
			response, err := UpdateEmergencyContact(database, testContactID, nil)
			So(err, ShouldBeNil)
			So(response, ShouldResemble, dto.SaveEmergencyContactResponse{})
		})

		Convey("With empty contact id", func() {
			emergencyContactDTO := dto.EmergencyContact{
				HeartbeatTypes: []datatypes.HeartbeatType{datatypes.HeartbeatNotifier},
			}
			database.EXPECT().SaveEmergencyContact(testEmergencyContact).Return(nil)

			response, err := UpdateEmergencyContact(database, testContactID, &emergencyContactDTO)
			So(err, ShouldBeNil)
			So(response, ShouldResemble, dto.SaveEmergencyContactResponse{
				ContactID: testContactID,
			})
		})

		Convey("With full filled emergency contact dto", func() {
			database.EXPECT().SaveEmergencyContact(testEmergencyContact).Return(nil)

			response, err := UpdateEmergencyContact(database, testContactID, &testEmergencyContactDTO)
			So(err, ShouldBeNil)
			So(response, ShouldResemble, dto.SaveEmergencyContactResponse{
				ContactID: testContactID,
			})
		})

		Convey("With database error", func() {
			dbErr := errors.New("update emergency contact error")
			database.EXPECT().SaveEmergencyContact(testEmergencyContact).Return(dbErr)

			response, err := UpdateEmergencyContact(database, testContactID, &testEmergencyContactDTO)
			So(err, ShouldResemble, api.ErrorInternalServer(dbErr))
			So(response, ShouldResemble, dto.SaveEmergencyContactResponse{})
		})
	})
}

func TestRemoveEmergencyContact(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	defer mockCtrl.Finish()

	Convey("Test RemoveEmergencyContact", t, func() {
		Convey("Successfully removed emergency contact", func() {
			database.EXPECT().RemoveEmergencyContact(testContactID).Return(nil)

			err := RemoveEmergencyContact(database, testContactID)
			So(err, ShouldBeNil)
		})

		Convey("With database error", func() {
			dbErr := errors.New("remove emergency contact error")
			database.EXPECT().RemoveEmergencyContact(testContactID).Return(dbErr)

			err := RemoveEmergencyContact(database, testContactID)
			So(err, ShouldResemble, api.ErrorInternalServer(dbErr))
		})
	})
}
