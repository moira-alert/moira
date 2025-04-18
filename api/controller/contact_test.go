package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/dto"
	"github.com/moira-alert/moira/database"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/mock/gomock"
)

func TestGetAllContacts(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)

	Convey("Error get all contacts", t, func() {
		expected := fmt.Errorf("oooops! Can not get all contacts")
		dataBase.EXPECT().GetAllContacts().Return(nil, expected)
		contacts, err := GetAllContacts(dataBase)
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
		So(contacts, ShouldBeNil)
	})

	Convey("Get contacts", t, func() {
		contacts := []*moira.ContactData{
			{
				ID:    uuid.Must(uuid.NewV4()).String(),
				Type:  "mail",
				User:  "user1",
				Value: "good@mail.com",
			},
			{
				ID:    uuid.Must(uuid.NewV4()).String(),
				Type:  "pushover",
				User:  "user2",
				Value: "ggg1",
			},
		}
		dataBase.EXPECT().GetAllContacts().Return(contacts, nil)
		actual, err := GetAllContacts(dataBase)
		So(err, ShouldBeNil)

		expectedContacts := []dto.TeamContact{
			{
				ID:    contacts[0].ID,
				Type:  "mail",
				User:  "user1",
				Value: "good@mail.com",
			},
			{
				ID:    contacts[1].ID,
				Type:  "pushover",
				User:  "user2",
				Value: "ggg1",
			},
		}
		So(actual, ShouldResemble, &dto.ContactList{List: expectedContacts})
	})

	Convey("No contacts", t, func() {
		dataBase.EXPECT().GetAllContacts().Return(make([]*moira.ContactData, 0), nil)
		contacts, err := GetAllContacts(dataBase)
		So(err, ShouldBeNil)
		So(contacts, ShouldResemble, &dto.ContactList{List: make([]dto.TeamContact, 0)})
	})
}

func TestGetContactById(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)

	Convey("Get contact by id should be success", t, func() {
		contact := moira.ContactData{
			ID:    uuid.Must(uuid.NewV4()).String(),
			Name:  "awesome_name",
			Type:  "slack",
			User:  "awesome_moira_user",
			Value: "awesome_moira_user@gmail.com",
		}

		dataBase.EXPECT().GetContact(contact.ID).Return(contact, nil)
		actual, err := GetContactById(dataBase, contact.ID)
		So(err, ShouldBeNil)
		So(actual,
			ShouldResemble,
			&dto.Contact{
				ID:    contact.ID,
				Name:  contact.Name,
				Type:  contact.Type,
				User:  contact.User,
				Value: contact.Value,
			})
	})

	Convey("Get contact with invalid or unexisting guid id should be empty json", t, func() {
		const invalidId = "invalidID"

		dataBase.EXPECT().GetContact(invalidId).Return(moira.ContactData{}, nil)
		actual, err := GetContactById(dataBase, invalidId)
		So(err, ShouldBeNil)
		So(actual, ShouldResemble, &dto.Contact{})
	})

	Convey("Error to fetch contact from db should rise api error", t, func() {
		const contactID = "no-matter-what-id-is-there"

		emptyContact := moira.ContactData{}
		dbError := fmt.Errorf("some db internal error here")

		dataBase.EXPECT().GetContact(contactID).Return(emptyContact, dbError)
		contact, err := GetContactById(dataBase, contactID)
		So(err, ShouldResemble, api.ErrorInternalServer(dbError))
		So(contact, ShouldBeNil)
	})
}

func TestCreateContact(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	defer mockCtrl.Finish()

	const (
		userLogin             = "user"
		teamID                = "team"
		contactType           = "mail"
		notAllowedContactType = "mattermost"
		contactValue          = "some@mail.com"
	)

	auth := &api.Authorization{
		Enabled: true,
		AllowedContactTypes: map[string]struct{}{
			contactType: {},
		},
	}

	contactsTemplate := []api.WebContact{
		{
			ContactType:     contactType,
			ValidationRegex: "@mail.com",
		},
	}

	Convey("Create for user", t, func() {
		Convey("Success", func() {
			contact := &dto.Contact{
				Value: contactValue,
				Type:  contactType,
			}

			dataBase.EXPECT().SaveContact(gomock.Any()).Return(nil)
			err := CreateContact(dataBase, auth, contactsTemplate, contact, userLogin, "")
			So(err, ShouldBeNil)
			So(contact.User, ShouldResemble, userLogin)
		})

		Convey("Success with id", func() {
			contact := dto.Contact{
				ID:    uuid.Must(uuid.NewV4()).String(),
				Value: contactValue,
				Type:  contactType,
			}
			expectedContact := moira.ContactData{
				ID:    contact.ID,
				Value: contact.Value,
				Type:  contact.Type,
				User:  userLogin,
			}

			dataBase.EXPECT().GetContact(contact.ID).Return(moira.ContactData{}, database.ErrNil)
			dataBase.EXPECT().SaveContact(&expectedContact).Return(nil)
			err := CreateContact(dataBase, auth, contactsTemplate, &contact, userLogin, "")
			So(err, ShouldBeNil)
			So(contact.User, ShouldResemble, userLogin)
			So(contact.ID, ShouldResemble, contact.ID)
		})

		Convey("Contact exists by id", func() {
			contact := &dto.Contact{
				ID:    uuid.Must(uuid.NewV4()).String(),
				Value: contactValue,
				Type:  contactType,
			}
			dataBase.EXPECT().GetContact(contact.ID).Return(moira.ContactData{}, nil)
			err := CreateContact(dataBase, auth, contactsTemplate, contact, userLogin, "")
			So(err, ShouldResemble, api.ErrorInvalidRequest(fmt.Errorf("contact with this ID already exists")))
		})

		Convey("Error get contact", func() {
			contact := &dto.Contact{
				ID:    uuid.Must(uuid.NewV4()).String(),
				Value: contactValue,
				Type:  contactType,
			}
			err := fmt.Errorf("oooops! Can not write contact")
			dataBase.EXPECT().GetContact(contact.ID).Return(moira.ContactData{}, err)
			expected := CreateContact(dataBase, auth, contactsTemplate, contact, userLogin, "")
			So(expected, ShouldResemble, api.ErrorInternalServer(err))
		})

		contactsTemplate = []api.WebContact{
			{
				ContactType:     contactType,
				ValidationRegex: "@yandex.ru",
			},
		}

		Convey("Error invalid contact value", func() {
			contact := &dto.Contact{
				Value: contactValue,
				Type:  contactType,
			}
			expectedErr := api.ErrorInvalidRequest(fmt.Errorf("contact value doesn't match regex: '%s'", "@yandex.ru"))
			err := CreateContact(dataBase, auth, contactsTemplate, contact, userLogin, "")
			So(err, ShouldResemble, expectedErr)
		})

		contactsTemplate = []api.WebContact{
			{
				ContactType:     contactType,
				ValidationRegex: "@mail.com",
			},
		}

		Convey("Error create now allowed contact", func() {
			contact := &dto.Contact{
				ID:    uuid.Must(uuid.NewV4()).String(),
				Value: contactValue,
				Type:  notAllowedContactType,
			}
			expectedErr := api.ErrorInvalidRequest(ErrNotAllowedContactType)
			err := CreateContact(dataBase, auth, contactsTemplate, contact, userLogin, "")
			So(err, ShouldResemble, expectedErr)
		})

		Convey("Successfully create not allowed contact with disabled auth", func() {
			auth.Enabled = false
			defer func() {
				auth.Enabled = true
			}()

			contact := &dto.Contact{
				ID:    uuid.Must(uuid.NewV4()).String(),
				Value: contactValue,
				Type:  notAllowedContactType,
			}

			expectedContact := moira.ContactData{
				ID:    contact.ID,
				Value: contact.Value,
				Type:  contact.Type,
				User:  userLogin,
			}

			dataBase.EXPECT().GetContact(contact.ID).Return(moira.ContactData{}, database.ErrNil)
			dataBase.EXPECT().SaveContact(&expectedContact).Return(nil)

			err := CreateContact(dataBase, auth, contactsTemplate, contact, userLogin, "")
			So(err, ShouldBeNil)
		})

		Convey("Error save contact", func() {
			contact := &dto.Contact{
				Value: contactValue,
				Type:  contactType,
			}
			err := fmt.Errorf("oooops! Can not write contact")
			dataBase.EXPECT().SaveContact(gomock.Any()).Return(err)
			expected := CreateContact(dataBase, auth, contactsTemplate, contact, userLogin, "")
			So(expected, ShouldResemble, &api.ErrorResponse{
				ErrorText:      err.Error(),
				HTTPStatusCode: http.StatusInternalServerError,
				StatusText:     "Internal Server Error",
				Err:            err,
			})
		})
	})

	Convey("Create for team", t, func() {
		Convey("Success", func() {
			contact := &dto.Contact{
				Value: contactValue,
				Type:  contactType,
			}

			dataBase.EXPECT().SaveContact(gomock.Any()).Return(nil)
			err := CreateContact(dataBase, auth, contactsTemplate, contact, "", teamID)
			So(err, ShouldBeNil)
			So(contact.TeamID, ShouldResemble, teamID)
		})

		Convey("Success with id", func() {
			contact := dto.Contact{
				ID:    uuid.Must(uuid.NewV4()).String(),
				Value: contactValue,
				Type:  contactType,
			}
			expectedContact := moira.ContactData{
				ID:    contact.ID,
				Value: contact.Value,
				Type:  contact.Type,
				Team:  teamID,
			}

			dataBase.EXPECT().GetContact(contact.ID).Return(moira.ContactData{}, database.ErrNil)
			dataBase.EXPECT().SaveContact(&expectedContact).Return(nil)
			err := CreateContact(dataBase, auth, contactsTemplate, &contact, "", teamID)
			So(err, ShouldBeNil)
			So(contact.TeamID, ShouldResemble, teamID)
			So(contact.ID, ShouldResemble, contact.ID)
		})

		Convey("Success with custom name", func() {
			contact := dto.Contact{
				ID:    uuid.Must(uuid.NewV4()).String(),
				Value: contactValue,
				Type:  contactType,
				Name:  "some-name",
			}
			expectedContact := moira.ContactData{
				ID:    contact.ID,
				Value: contact.Value,
				Type:  contact.Type,
				Name:  contact.Name,
				Team:  teamID,
			}

			dataBase.EXPECT().GetContact(contact.ID).Return(moira.ContactData{}, database.ErrNil)
			dataBase.EXPECT().SaveContact(&expectedContact).Return(nil)
			err := CreateContact(dataBase, auth, contactsTemplate, &contact, "", teamID)
			So(err, ShouldBeNil)
			So(contact.TeamID, ShouldResemble, teamID)
			So(contact.Name, ShouldResemble, expectedContact.Name)
		})

		Convey("Contact exists by id", func() {
			contact := &dto.Contact{
				ID:    uuid.Must(uuid.NewV4()).String(),
				Value: contactValue,
				Type:  contactType,
			}
			dataBase.EXPECT().GetContact(contact.ID).Return(moira.ContactData{}, nil)
			err := CreateContact(dataBase, auth, contactsTemplate, contact, "", teamID)
			So(err, ShouldResemble, api.ErrorInvalidRequest(fmt.Errorf("contact with this ID already exists")))
		})

		Convey("Error get contact", func() {
			contact := &dto.Contact{
				ID:    uuid.Must(uuid.NewV4()).String(),
				Value: contactValue,
				Type:  contactType,
			}
			err := fmt.Errorf("oooops! Can not write contact")
			dataBase.EXPECT().GetContact(contact.ID).Return(moira.ContactData{}, err)
			expected := CreateContact(dataBase, auth, contactsTemplate, contact, "", teamID)
			So(expected, ShouldResemble, api.ErrorInternalServer(err))
		})

		Convey("Error create not allowed contact", func() {
			contact := &dto.Contact{
				ID:    uuid.Must(uuid.NewV4()).String(),
				Value: contactValue,
				Type:  notAllowedContactType,
			}
			expectedErr := api.ErrorInvalidRequest(ErrNotAllowedContactType)
			err := CreateContact(dataBase, auth, contactsTemplate, contact, "", teamID)
			So(err, ShouldResemble, expectedErr)
		})

		Convey("Successfully create not allowed contact with disabled auth", func() {
			auth.Enabled = false
			defer func() {
				auth.Enabled = true
			}()

			contact := &dto.Contact{
				ID:    uuid.Must(uuid.NewV4()).String(),
				Value: contactValue,
				Type:  notAllowedContactType,
			}

			expectedContact := moira.ContactData{
				ID:    contact.ID,
				Value: contact.Value,
				Type:  contact.Type,
				Team:  teamID,
			}

			dataBase.EXPECT().GetContact(contact.ID).Return(moira.ContactData{}, database.ErrNil)
			dataBase.EXPECT().SaveContact(&expectedContact).Return(nil)

			err := CreateContact(dataBase, auth, contactsTemplate, contact, "", teamID)
			So(err, ShouldBeNil)
		})

		Convey("Error save contact", func() {
			contact := &dto.Contact{
				Value: contactValue,
				Type:  contactType,
			}
			err := fmt.Errorf("oooops! Can not write contact")
			dataBase.EXPECT().SaveContact(gomock.Any()).Return(err)
			expected := CreateContact(dataBase, auth, contactsTemplate, contact, "", teamID)
			So(expected, ShouldResemble, &api.ErrorResponse{
				ErrorText:      err.Error(),
				HTTPStatusCode: http.StatusInternalServerError,
				StatusText:     "Internal Server Error",
				Err:            err,
			})
		})
	})
}

func TestAdminsCreatesContact(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	defer mockCtrl.Finish()

	const (
		userLogin             = "user"
		adminLogin            = "admin"
		contactType           = "mail"
		notAllowedContactType = "mattermost"
		contactValue          = "some@mail.com"
	)

	auth := &api.Authorization{
		Enabled:   true,
		AdminList: map[string]struct{}{adminLogin: {}},
		AllowedContactTypes: map[string]struct{}{
			contactType: {},
		},
	}

	contactsTemplate := []api.WebContact{}

	Convey("Create for user", t, func() {
		Convey("The same user", func() {
			contact := &dto.Contact{
				Value: contactValue,
				Type:  contactType,
				User:  userLogin,
			}

			dataBase.EXPECT().SaveContact(gomock.Any()).Return(nil)
			err := CreateContact(dataBase, auth, contactsTemplate, contact, userLogin, "")
			So(err, ShouldBeNil)
			So(contact.User, ShouldResemble, userLogin)
		})

		Convey("The same user by admin", func() {
			contact := &dto.Contact{
				Value: contactValue,
				Type:  contactType,
				User:  adminLogin,
			}

			dataBase.EXPECT().SaveContact(gomock.Any()).Return(nil)
			err := CreateContact(dataBase, auth, contactsTemplate, contact, adminLogin, "")
			So(err, ShouldBeNil)
			So(contact.User, ShouldResemble, adminLogin)
		})

		Convey("Non admin can not create contact for other user", func() {
			contact := &dto.Contact{
				Value: contactValue,
				Type:  contactType,
				User:  adminLogin,
			}

			dataBase.EXPECT().SaveContact(gomock.Any()).Return(nil)
			err := CreateContact(dataBase, auth, contactsTemplate, contact, userLogin, "")
			So(err, ShouldBeNil)
			So(contact.User, ShouldResemble, userLogin)
		})

		Convey("Admin can create contact for other user", func() {
			contact := &dto.Contact{
				Value: contactValue,
				Type:  contactType,
				User:  userLogin,
			}

			dataBase.EXPECT().SaveContact(gomock.Any()).Return(nil)
			err := CreateContact(dataBase, auth, contactsTemplate, contact, adminLogin, "")
			So(err, ShouldBeNil)
			So(contact.User, ShouldResemble, userLogin)
		})

		Convey("Admin can create not allowed contact", func() {
			contact := &dto.Contact{
				Value: contactValue,
				Type:  notAllowedContactType,
				User:  userLogin,
			}

			dataBase.EXPECT().SaveContact(gomock.Any()).Return(nil)
			err := CreateContact(dataBase, auth, contactsTemplate, contact, adminLogin, "")
			So(err, ShouldBeNil)
			So(contact.User, ShouldResemble, userLogin)
		})
	})
}

func TestUpdateContact(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	defer mockCtrl.Finish()

	const (
		userLogin             = "user"
		teamID                = "team"
		contactType           = "mail"
		contactValue          = "some@mail.com"
		notAllowedContactType = "mattermost"
	)

	auth := &api.Authorization{
		Enabled: true,
		AllowedContactTypes: map[string]struct{}{
			contactType: {},
		},
	}

	contactsTemplate := []api.WebContact{
		{
			ContactType:     contactType,
			ValidationRegex: "@mail.com",
		},
	}

	Convey("User update", t, func() {
		Convey("Success", func() {
			contactDTO := dto.Contact{
				Value: contactValue,
				Name:  "some-name",
				Type:  contactType,
			}
			contactID := uuid.Must(uuid.NewV4()).String()
			contact := moira.ContactData{
				Value: contactDTO.Value,
				Type:  contactDTO.Type,
				Name:  contactDTO.Name,
				ID:    contactID,
				User:  userLogin,
			}
			dataBase.EXPECT().SaveContact(&contact).Return(nil)
			expectedContact, err := UpdateContact(dataBase, auth, contactsTemplate, contactDTO, moira.ContactData{ID: contactID, User: userLogin})
			So(err, ShouldBeNil)
			So(expectedContact.User, ShouldResemble, userLogin)
			So(expectedContact.ID, ShouldResemble, contactID)
			So(expectedContact.Name, ShouldResemble, contactDTO.Name)
		})

		Convey("Success with rewrite user", func() {
			newUser := "testUser"
			contactDTO := dto.Contact{
				Value: contactValue,
				Name:  "some-name",
				Type:  contactType,
				User:  newUser,
			}
			contactID := uuid.Must(uuid.NewV4()).String()
			contact := moira.ContactData{
				Value: contactDTO.Value,
				Type:  contactDTO.Type,
				Name:  contactDTO.Name,
				ID:    contactID,
				User:  newUser,
			}
			dataBase.EXPECT().SaveContact(&contact).Return(nil)
			expectedContact, err := UpdateContact(dataBase, auth, contactsTemplate, contactDTO, moira.ContactData{ID: contactID, User: userLogin})
			So(err, ShouldBeNil)
			So(expectedContact.User, ShouldResemble, newUser)
			So(expectedContact.ID, ShouldResemble, contactID)
			So(expectedContact.Name, ShouldResemble, contactDTO.Name)
		})

		Convey("Error update not allowed contact", func() {
			contactDTO := dto.Contact{
				Value: contactValue,
				Type:  notAllowedContactType,
			}
			expectedErr := api.ErrorInvalidRequest(ErrNotAllowedContactType)
			contactID := uuid.Must(uuid.NewV4()).String()
			expectedContact, err := UpdateContact(dataBase, auth, contactsTemplate, contactDTO, moira.ContactData{ID: contactID, User: userLogin})
			So(err, ShouldResemble, expectedErr)
			So(expectedContact.User, ShouldResemble, contactDTO.User)
			So(expectedContact.ID, ShouldResemble, contactDTO.ID)
			So(expectedContact.Name, ShouldResemble, contactDTO.Name)
		})

		contactsTemplate = []api.WebContact{
			{
				ContactType:     contactType,
				ValidationRegex: "@yandex.ru",
			},
		}

		Convey("Error invalid contact value", func() {
			contactDTO := dto.Contact{
				Value: contactValue,
				Type:  contactType,
			}
			expectedErr := api.ErrorInvalidRequest(fmt.Errorf("contact value doesn't match regex: '%s'", "@yandex.ru"))
			contactID := uuid.Must(uuid.NewV4()).String()
			expectedContact, err := UpdateContact(dataBase, auth, contactsTemplate, contactDTO, moira.ContactData{ID: contactID, User: userLogin})
			So(err, ShouldResemble, expectedErr)
			So(expectedContact.User, ShouldResemble, contactDTO.User)
			So(expectedContact.ID, ShouldResemble, contactDTO.ID)
			So(expectedContact.Name, ShouldResemble, contactDTO.Name)
		})

		contactsTemplate = []api.WebContact{
			{
				ContactType:     contactType,
				ValidationRegex: "@mail.com",
			},
		}

		Convey("Successfully update not allowed contact with disabled auth", func() {
			auth.Enabled = false
			defer func() {
				auth.Enabled = true
			}()

			contactDTO := dto.Contact{
				Value: contactValue,
				Name:  "some-name",
				Type:  notAllowedContactType,
			}
			contactID := uuid.Must(uuid.NewV4()).String()
			contact := moira.ContactData{
				Value: contactDTO.Value,
				Type:  contactDTO.Type,
				Name:  contactDTO.Name,
				ID:    contactID,
				User:  userLogin,
			}

			dataBase.EXPECT().SaveContact(&contact).Return(nil)
			expectedContact, err := UpdateContact(dataBase, auth, contactsTemplate, contactDTO, moira.ContactData{ID: contactID, User: userLogin})
			So(err, ShouldBeNil)
			So(expectedContact.User, ShouldResemble, userLogin)
			So(expectedContact.ID, ShouldResemble, contactID)
			So(expectedContact.Name, ShouldResemble, contactDTO.Name)
		})

		Convey("Error save", func() {
			contactDTO := dto.Contact{
				Value: contactValue,
				Type:  contactType,
			}
			contactID := uuid.Must(uuid.NewV4()).String()
			contact := moira.ContactData{
				Value: contactDTO.Value,
				Type:  contactDTO.Type,
				ID:    contactID,
				User:  userLogin,
			}
			err := fmt.Errorf("oooops")
			dataBase.EXPECT().SaveContact(&contact).Return(err)
			expectedContact, actual := UpdateContact(dataBase, auth, contactsTemplate, contactDTO, contact)
			So(actual, ShouldResemble, api.ErrorInternalServer(err))
			So(expectedContact.User, ShouldResemble, contactDTO.User)
			So(expectedContact.ID, ShouldResemble, contactDTO.ID)
		})
	})

	Convey("Team update", t, func() {
		Convey("Success", func() {
			contactDTO := dto.Contact{
				Value: contactValue,
				Type:  contactType,
			}
			contactID := uuid.Must(uuid.NewV4()).String()
			contact := moira.ContactData{
				Value: contactDTO.Value,
				Type:  contactDTO.Type,
				ID:    contactID,
				Team:  teamID,
			}
			dataBase.EXPECT().SaveContact(&contact).Return(nil)
			expectedContact, err := UpdateContact(dataBase, auth, contactsTemplate, contactDTO, moira.ContactData{ID: contactID, Team: teamID})
			So(err, ShouldBeNil)
			So(expectedContact.TeamID, ShouldResemble, teamID)
			So(expectedContact.ID, ShouldResemble, contactID)
		})

		Convey("Success with rewrite team", func() {
			newTeam := "testTeam"
			contactDTO := dto.Contact{
				Value:  contactValue,
				Type:   contactType,
				TeamID: newTeam,
			}
			contactID := uuid.Must(uuid.NewV4()).String()
			contact := moira.ContactData{
				Value: contactDTO.Value,
				Type:  contactDTO.Type,
				ID:    contactID,
				Team:  newTeam,
			}
			dataBase.EXPECT().SaveContact(&contact).Return(nil)
			expectedContact, err := UpdateContact(dataBase, auth, contactsTemplate, contactDTO, moira.ContactData{ID: contactID, Team: teamID})
			So(err, ShouldBeNil)
			So(expectedContact.TeamID, ShouldResemble, newTeam)
			So(expectedContact.ID, ShouldResemble, contactID)
		})

		Convey("Error save", func() {
			contactDTO := dto.Contact{
				Value: contactValue,
				Type:  contactType,
			}
			contactID := uuid.Must(uuid.NewV4()).String()
			contact := moira.ContactData{
				Value: contactDTO.Value,
				Type:  contactDTO.Type,
				ID:    contactID,
				Team:  teamID,
			}
			err := fmt.Errorf("oooops")
			dataBase.EXPECT().SaveContact(&contact).Return(err)
			expectedContact, actual := UpdateContact(dataBase, auth, contactsTemplate, contactDTO, contact)
			So(actual, ShouldResemble, api.ErrorInternalServer(err))
			So(expectedContact.TeamID, ShouldResemble, contactDTO.TeamID)
			So(expectedContact.ID, ShouldResemble, contactDTO.ID)
		})
	})
}

func TestIsAllowedContactType(t *testing.T) {
	const (
		admin                 = "admin"
		user                  = "user"
		allowedContactType    = "slack"
		notAllowedContactType = "mattermost"
	)

	auth := &api.Authorization{
		Enabled: true,
		AdminList: map[string]struct{}{
			admin: {},
		},
		AllowedContactTypes: map[string]struct{}{
			allowedContactType: {},
		},
	}

	Convey("Test isAllowedContactType", t, func() {
		Convey("Test with user and allowed contact type", func() {
			isAllowed := isAllowedToUseContactType(auth, user, allowedContactType)
			So(isAllowed, ShouldBeTrue)
		})

		Convey("Test with user and not allowed contact type", func() {
			isAllowed := isAllowedToUseContactType(auth, user, notAllowedContactType)
			So(isAllowed, ShouldBeFalse)
		})

		Convey("Test with admin and allowed contact type", func() {
			isAllowed := isAllowedToUseContactType(auth, admin, allowedContactType)
			So(isAllowed, ShouldBeTrue)
		})

		Convey("Test with admin and not allowed contact type", func() {
			isAllowed := isAllowedToUseContactType(auth, admin, notAllowedContactType)
			So(isAllowed, ShouldBeTrue)
		})

		Convey("Test with disabled auth and not allowed contact type", func() {
			auth.Enabled = false
			isAllowed := isAllowedToUseContactType(auth, admin, notAllowedContactType)
			So(isAllowed, ShouldBeTrue)
		})

		Convey("Test with disabled auth and allowed contact type", func() {
			auth.Enabled = false
			isAllowed := isAllowedToUseContactType(auth, admin, allowedContactType)
			So(isAllowed, ShouldBeTrue)
		})
	})
}

func TestRemoveContact(t *testing.T) {
	const userLogin = "user"

	const teamID = "team"

	contactID := uuid.Must(uuid.NewV4()).String()

	Convey("Delete user contact", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)

		Convey("Without subscriptions", func() {
			dataBase.EXPECT().GetUserSubscriptionIDs(userLogin).Return(make([]string, 0), nil)
			dataBase.EXPECT().GetSubscriptions(make([]string, 0)).Return(make([]*moira.SubscriptionData, 0), nil)
			dataBase.EXPECT().RemoveContact(contactID).Return(nil)
			err := RemoveContact(dataBase, contactID, userLogin, "")
			So(err, ShouldBeNil)
		})

		Convey("Without contact subscriptions", func() {
			subscription := &moira.SubscriptionData{
				Contacts: []string{uuid.Must(uuid.NewV4()).String()},
				ID:       uuid.Must(uuid.NewV4()).String(),
			}

			dataBase.EXPECT().GetUserSubscriptionIDs(userLogin).Return([]string{subscription.ID}, nil)
			dataBase.EXPECT().GetSubscriptions([]string{subscription.ID}).Return([]*moira.SubscriptionData{subscription}, nil)
			dataBase.EXPECT().RemoveContact(contactID).Return(nil)
			err := RemoveContact(dataBase, contactID, userLogin, "")
			So(err, ShouldBeNil)
		})

		Convey("Error tests", func() {
			Convey("GetUserSubscriptionIDs", func() {
				expectedError := fmt.Errorf("oooops! Can not read user subscription ids")
				dataBase.EXPECT().GetUserSubscriptionIDs(userLogin).Return(nil, expectedError)
				err := RemoveContact(dataBase, contactID, userLogin, "")
				So(err, ShouldResemble, api.ErrorInternalServer(expectedError))
			})
			Convey("GetSubscriptions", func() {
				expectedError := fmt.Errorf("oooops! Can not read user subscriptions")

				dataBase.EXPECT().GetUserSubscriptionIDs(userLogin).Return(make([]string, 0), nil)
				dataBase.EXPECT().GetSubscriptions(make([]string, 0)).Return(nil, expectedError)
				err := RemoveContact(dataBase, contactID, userLogin, "")
				So(err, ShouldResemble, api.ErrorInternalServer(expectedError))
			})
			Convey("Subscription has contact", func() {
				subscription := moira.SubscriptionData{
					Contacts: []string{contactID},
					ID:       uuid.Must(uuid.NewV4()).String(),
					Tags:     []string{"Tag1", "Tag2"},
				}
				subscriptionSubstring := fmt.Sprintf("%s (tags: %s)", subscription.ID, strings.Join(subscription.Tags, ", "))
				expectedError := fmt.Errorf("this contact is being used in following subscriptions: %s", subscriptionSubstring)

				dataBase.EXPECT().GetUserSubscriptionIDs(userLogin).Return([]string{subscription.ID}, nil)
				dataBase.EXPECT().GetSubscriptions([]string{subscription.ID}).Return([]*moira.SubscriptionData{&subscription}, nil)
				err := RemoveContact(dataBase, contactID, userLogin, "")
				So(err, ShouldResemble, api.ErrorInvalidRequest(expectedError))
			})
		})
	})

	Convey("Delete team contact", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)

		Convey("Without subscriptions", func() {
			dataBase.EXPECT().GetTeamSubscriptionIDs(teamID).Return(make([]string, 0), nil)
			dataBase.EXPECT().GetSubscriptions(make([]string, 0)).Return(make([]*moira.SubscriptionData, 0), nil)
			dataBase.EXPECT().RemoveContact(contactID).Return(nil)
			err := RemoveContact(dataBase, contactID, "", teamID)
			So(err, ShouldBeNil)
		})

		Convey("Without contact subscriptions", func() {
			subscription := &moira.SubscriptionData{
				Contacts: []string{uuid.Must(uuid.NewV4()).String()},
				ID:       uuid.Must(uuid.NewV4()).String(),
			}

			dataBase.EXPECT().GetTeamSubscriptionIDs(teamID).Return([]string{subscription.ID}, nil)
			dataBase.EXPECT().GetSubscriptions([]string{subscription.ID}).Return([]*moira.SubscriptionData{subscription}, nil)
			dataBase.EXPECT().RemoveContact(contactID).Return(nil)
			err := RemoveContact(dataBase, contactID, "", teamID)
			So(err, ShouldBeNil)
		})

		Convey("Error tests", func() {
			Convey("GetTeamSubscriptionIDs", func() {
				expectedError := fmt.Errorf("oooops! Can not read team subscription ids")
				dataBase.EXPECT().GetTeamSubscriptionIDs(teamID).Return(nil, expectedError)
				err := RemoveContact(dataBase, contactID, "", teamID)
				So(err, ShouldResemble, api.ErrorInternalServer(expectedError))
			})
			Convey("GetSubscriptions", func() {
				expectedError := fmt.Errorf("oooops! Can not read team subscriptions")

				dataBase.EXPECT().GetTeamSubscriptionIDs(teamID).Return(make([]string, 0), nil)
				dataBase.EXPECT().GetSubscriptions(make([]string, 0)).Return(nil, expectedError)
				err := RemoveContact(dataBase, contactID, "", teamID)
				So(err, ShouldResemble, api.ErrorInternalServer(expectedError))
			})
			Convey("Subscription has contact", func() {
				subscription := moira.SubscriptionData{
					Contacts: []string{contactID},
					ID:       uuid.Must(uuid.NewV4()).String(),
					Tags:     []string{"Tag1", "Tag2"},
				}
				subscriptionSubstring := fmt.Sprintf("%s (tags: %s)", subscription.ID, strings.Join(subscription.Tags, ", "))
				expectedError := fmt.Errorf("this contact is being used in following subscriptions: %s", subscriptionSubstring)

				dataBase.EXPECT().GetTeamSubscriptionIDs(teamID).Return([]string{subscription.ID}, nil)
				dataBase.EXPECT().GetSubscriptions([]string{subscription.ID}).Return([]*moira.SubscriptionData{&subscription}, nil)
				err := RemoveContact(dataBase, contactID, "", teamID)
				So(err, ShouldResemble, api.ErrorInvalidRequest(expectedError))
			})
		})
	})
}

func TestSendTestContactNotification(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	id := uuid.Must(uuid.NewV4()).String()

	Convey("Success", t, func() {
		dataBase.EXPECT().PushNotificationEvent(gomock.Any(), false).Return(nil)
		err := SendTestContactNotification(dataBase, id)
		So(err, ShouldBeNil)
	})

	Convey("Error", t, func() {
		expected := fmt.Errorf("oooops! Can not push event")
		dataBase.EXPECT().PushNotificationEvent(gomock.Any(), false).Return(expected)
		err := SendTestContactNotification(dataBase, id)
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
	})
}

func TestCheckUserPermissionsForContact(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	userLogin := uuid.Must(uuid.NewV4()).String()
	teamID := uuid.Must(uuid.NewV4()).String()
	id := uuid.Must(uuid.NewV4()).String()
	auth := &api.Authorization{}

	Convey("No contact", t, func() {
		dataBase.EXPECT().GetContact(id).Return(moira.ContactData{}, database.ErrNil)
		expectedContact, expected := CheckUserPermissionsForContact(dataBase, id, userLogin, auth)
		So(expected, ShouldResemble, api.ErrorNotFound(fmt.Sprintf("contact with ID '%s' does not exists", id)))
		So(expectedContact, ShouldResemble, moira.ContactData{})
	})

	Convey("Different user", t, func() {
		dataBase.EXPECT().GetContact(id).Return(moira.ContactData{User: "diffUser"}, nil)
		expectedContact, expected := CheckUserPermissionsForContact(dataBase, id, userLogin, auth)
		So(expected, ShouldResemble, api.ErrorForbidden("you are not permitted"))
		So(expectedContact, ShouldResemble, moira.ContactData{})
	})

	Convey("Has contact", t, func() {
		actualContact := moira.ContactData{ID: id, User: userLogin}
		dataBase.EXPECT().GetContact(id).Return(actualContact, nil)
		expectedContact, expected := CheckUserPermissionsForContact(dataBase, id, userLogin, auth)
		So(expected, ShouldBeNil)
		So(expectedContact, ShouldResemble, actualContact)
	})

	Convey("Error get contact", t, func() {
		err := fmt.Errorf("oooops! Can not read contact")
		dataBase.EXPECT().GetContact(id).Return(moira.ContactData{User: userLogin}, err)
		expectedContact, expected := CheckUserPermissionsForContact(dataBase, id, userLogin, auth)
		So(expected, ShouldResemble, api.ErrorInternalServer(err))
		So(expectedContact, ShouldResemble, moira.ContactData{})
	})

	Convey("Team contact", t, func() {
		Convey("User is in team", func() {
			expectedSub := moira.ContactData{ID: id, Team: teamID}
			dataBase.EXPECT().GetContact(id).Return(expectedSub, nil)
			dataBase.EXPECT().IsTeamContainUser(teamID, userLogin).Return(true, nil)
			actual, err := CheckUserPermissionsForContact(dataBase, id, userLogin, auth)
			So(err, ShouldBeNil)
			So(actual, ShouldResemble, expectedSub)
		})
		Convey("User is not in team", func() {
			dataBase.EXPECT().GetContact(id).Return(moira.ContactData{ID: id, Team: teamID}, nil)
			dataBase.EXPECT().IsTeamContainUser(teamID, userLogin).Return(false, nil)
			actual, err := CheckUserPermissionsForContact(dataBase, id, userLogin, auth)
			So(err, ShouldResemble, api.ErrorForbidden("you are not permitted"))
			So(actual, ShouldResemble, moira.ContactData{})
		})
		Convey("Error while checking user team", func() {
			errReturned := errors.New("test error")

			dataBase.EXPECT().GetContact(id).Return(moira.ContactData{ID: id, Team: teamID}, nil)
			dataBase.EXPECT().IsTeamContainUser(teamID, userLogin).Return(false, errReturned)
			actual, err := CheckUserPermissionsForContact(dataBase, id, userLogin, auth)
			So(err, ShouldResemble, api.ErrorInternalServer(errReturned))
			So(actual, ShouldResemble, moira.ContactData{})
		})
	})
}

func TestCheckAdminPermissionsForContact(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	teamID := uuid.Must(uuid.NewV4()).String()
	id := uuid.Must(uuid.NewV4()).String()
	adminLogin := "admin_login"
	auth := &api.Authorization{Enabled: true, AdminList: map[string]struct{}{adminLogin: {}}}

	Convey("Same user", t, func() {
		expectedContact := moira.ContactData{ID: id, User: adminLogin}
		dataBase.EXPECT().GetContact(id).Return(expectedContact, nil)
		actualContact, errorResponse := CheckUserPermissionsForContact(dataBase, id, adminLogin, auth)
		So(errorResponse, ShouldBeNil)
		So(actualContact, ShouldResemble, expectedContact)
	})

	Convey("Different user", t, func() {
		expectedContact := moira.ContactData{ID: id, User: "diffUser"}
		dataBase.EXPECT().GetContact(id).Return(expectedContact, nil)
		actualContact, errorResponse := CheckUserPermissionsForContact(dataBase, id, adminLogin, auth)
		So(errorResponse, ShouldBeNil)
		So(actualContact, ShouldResemble, expectedContact)
	})

	Convey("Team contact", t, func() {
		expectedContact := moira.ContactData{ID: id, Team: teamID}
		dataBase.EXPECT().GetContact(id).Return(expectedContact, nil)
		actualContact, errorResponse := CheckUserPermissionsForContact(dataBase, id, adminLogin, auth)
		So(errorResponse, ShouldBeNil)
		So(actualContact, ShouldResemble, expectedContact)
	})
}

func Test_isContactExists(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)

	const contactID = "testContact"
	contact := moira.ContactData{ID: contactID}

	Convey("isContactExists", t, func() {
		Convey("contact exists", func() {
			dataBase.EXPECT().GetContact(contactID).Return(contact, nil)
			actual, err := isContactExists(dataBase, contactID)
			So(actual, ShouldBeTrue)
			So(err, ShouldBeNil)
		})
		Convey("contact is not exist", func() {
			dataBase.EXPECT().GetContact(contactID).Return(moira.ContactData{}, database.ErrNil)
			actual, err := isContactExists(dataBase, contactID)
			So(actual, ShouldBeFalse)
			So(err, ShouldBeNil)
		})
		Convey("error returned", func() {
			errReturned := errors.New("some error")
			dataBase.EXPECT().GetContact(contactID).Return(moira.ContactData{}, errReturned)
			actual, err := isContactExists(dataBase, contactID)
			So(actual, ShouldBeFalse)
			So(err, ShouldResemble, errReturned)
		})
	})
}

func TestValidateContact(t *testing.T) {
	const (
		contactType  = "phone"
		contactValue = "+79998887766"
	)

	Convey("Test validateContact", t, func() {
		contact := moira.ContactData{
			Type:  contactType,
			Value: contactValue,
		}

		Convey("With empty contactsTemplate", func() {
			contactsTemplate := []api.WebContact{}

			err := validateContact(contactsTemplate, contact)
			So(err, ShouldBeNil)
		})

		Convey("With not matched regex pattern", func() {
			contactsTemplate := []api.WebContact{
				{
					ContactType:     contactType,
					ValidationRegex: "^9\\d{9}$",
				},
			}

			notMatchedErr := fmt.Errorf("contact value doesn't match regex: '%s'", "^9\\d{9}$")

			err := validateContact(contactsTemplate, contact)
			So(err, ShouldResemble, notMatchedErr)
		})

		Convey("With matched regex pattern", func() {
			contactsTemplate := []api.WebContact{
				{
					ContactType:     contactType,
					ValidationRegex: `^\+79\d{9}$`,
				},
			}

			err := validateContact(contactsTemplate, contact)
			So(err, ShouldBeNil)
		})
	})
}

func TestGetContactNoisiness(t *testing.T) {
	Convey("Test get contact noisiness", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)

		const (
			allTimeFrom      = "-inf"
			allTimeTo        = "+inf"
			zeroPage         = int64(0)
			allEventsSize    = int64(-1)
			defaultSortOrder = api.DescSortOrder
		)

		contacts := []*moira.ContactData{
			{
				ID: "contactID1",
			},
			{
				ID: "contactID2",
			},
			{
				ID: "contactID3",
			},
		}

		Convey("On error, when getting all contacts, return 500", func() {
			someErr := errors.New("error from db")

			dataBase.EXPECT().GetAllContacts().Return(nil, someErr).Times(1)

			gotDTO, gotErrRsp := GetContactNoisiness(dataBase, zeroPage, allEventsSize, allTimeFrom, allTimeTo, defaultSortOrder)
			So(gotDTO, ShouldBeNil)
			So(gotErrRsp, ShouldResemble, api.ErrorInternalServer(someErr))
		})

		Convey("On error, when getting events count for contacts, return 500", func() {
			someErr := errors.New("error from db")

			dataBase.EXPECT().GetAllContacts().Return(contacts, nil).Times(1)
			dataBase.EXPECT().
				CountEventsInNotificationHistory([]string{"contactID1", "contactID2", "contactID3"}, allTimeFrom, allTimeTo).
				Return(nil, someErr).
				Times(1)

			gotDTO, gotErrRsp := GetContactNoisiness(dataBase, zeroPage, allEventsSize, allTimeFrom, allTimeTo, defaultSortOrder)
			So(gotDTO, ShouldBeNil)
			So(gotErrRsp, ShouldResemble, api.ErrorInternalServer(someErr))
		})

		Convey("No errors from db, noisiness got and sorted", func() {
			Convey("with desc sort order", func() {
				dataBase.EXPECT().GetAllContacts().Return(contacts, nil).Times(1)
				dataBase.EXPECT().CountEventsInNotificationHistory(
					[]string{"contactID1", "contactID2", "contactID3"},
					allTimeFrom,
					allTimeTo).
					Return([]*moira.ContactIDWithNotificationCount{
						{ID: "contactID1", Count: 2},
						{ID: "contactID2", Count: 3},
						{ID: "contactID3", Count: 1},
					}, nil).Times(1)

				gotDTO, gotErrRsp := GetContactNoisiness(dataBase, zeroPage, allEventsSize, allTimeFrom, allTimeTo, api.DescSortOrder)
				So(gotDTO, ShouldResemble, &dto.ContactNoisinessList{
					Page:  zeroPage,
					Size:  allEventsSize,
					Total: 3,
					List: []*dto.ContactNoisiness{
						{
							Contact:     dto.NewContact(*contacts[1]),
							EventsCount: 3,
						},
						{
							Contact:     dto.NewContact(*contacts[0]),
							EventsCount: 2,
						},
						{
							Contact:     dto.NewContact(*contacts[2]),
							EventsCount: 1,
						},
					},
				})
				So(gotErrRsp, ShouldBeNil)
			})

			Convey("with asc sort order", func() {
				dataBase.EXPECT().GetAllContacts().Return(contacts, nil).Times(1)
				dataBase.EXPECT().CountEventsInNotificationHistory(
					[]string{"contactID1", "contactID2", "contactID3"},
					allTimeFrom,
					allTimeTo).
					Return([]*moira.ContactIDWithNotificationCount{
						{ID: "contactID1", Count: 2},
						{ID: "contactID2", Count: 3},
						{ID: "contactID3", Count: 1},
					}, nil).Times(1)

				gotDTO, gotErrRsp := GetContactNoisiness(dataBase, zeroPage, allEventsSize, allTimeFrom, allTimeTo, api.AscSortOrder)
				So(gotDTO, ShouldResemble, &dto.ContactNoisinessList{
					Page:  zeroPage,
					Size:  allEventsSize,
					Total: 3,
					List: []*dto.ContactNoisiness{
						{
							Contact:     dto.NewContact(*contacts[2]),
							EventsCount: 1,
						},
						{
							Contact:     dto.NewContact(*contacts[0]),
							EventsCount: 2,
						},
						{
							Contact:     dto.NewContact(*contacts[1]),
							EventsCount: 3,
						},
					},
				})
				So(gotErrRsp, ShouldBeNil)
			})
		})
	})
}
