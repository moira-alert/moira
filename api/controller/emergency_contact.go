package controller

import (
	"errors"
	"fmt"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/dto"
	moiradb "github.com/moira-alert/moira/database"
	"github.com/moira-alert/moira/datatypes"
)

// ErrEmptyEmergencyContactID error occurring when user did not specify contact id.
var ErrEmptyEmergencyContactID = errors.New("emergency contact id can not be empty")

// GetEmergencyContacts a method for obtaining all emergency contacts.
func GetEmergencyContacts(database moira.Database) (*dto.EmergencyContactList, *api.ErrorResponse) {
	emergencyContacts, err := database.GetEmergencyContacts()
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}

	return dto.FromEmergencyContacts(emergencyContacts), nil
}

// GetEmergencyContact a method for obtaining a specific emergency contact.
func GetEmergencyContact(database moira.Database, contactID string) (*dto.EmergencyContact, *api.ErrorResponse) {
	emergencyContact, err := database.GetEmergencyContact(contactID)
	if err != nil {
		if errors.Is(err, moiradb.ErrNil) {
			return nil, api.ErrorNotFound(fmt.Sprintf("emergency contact with ID '%s' does not exists", contactID))
		}

		return nil, api.ErrorInternalServer(err)
	}

	emergencyContactDTO := dto.EmergencyContact(emergencyContact)

	return &emergencyContactDTO, nil
}

func verifyEmergencyContactAccess(
	database moira.Database,
	auth *api.Authorization,
	emergencyContact datatypes.EmergencyContact,
	userLogin string,
) *api.ErrorResponse {
	contact, err := database.GetContact(emergencyContact.ContactID)
	if err != nil {
		return api.ErrorInternalServer(err)
	}

	// Only admins are allowed to create an emergency contacts for other users
	if !auth.IsAdmin(userLogin) && contact.User != "" && contact.User != userLogin {
		return api.ErrorInvalidRequest(fmt.Errorf("cannot create an emergency contact using someone else's contact_id '%s'", emergencyContact.ContactID))
	}

	return nil
}

// CreateEmergencyContact a method for creating emergency contact.
func CreateEmergencyContact(
	database moira.Database,
	auth *api.Authorization,
	emergencyContactDTO *dto.EmergencyContact,
	userLogin string,
) (dto.SaveEmergencyContactResponse, *api.ErrorResponse) {
	if emergencyContactDTO == nil {
		return dto.SaveEmergencyContactResponse{}, nil
	}

	emergencyContact := datatypes.EmergencyContact(*emergencyContactDTO)
	if emergencyContact.ContactID == "" {
		return dto.SaveEmergencyContactResponse{}, api.ErrorInvalidRequest(ErrEmptyEmergencyContactID)
	}

	if err := verifyEmergencyContactAccess(database, auth, emergencyContact, userLogin); err != nil {
		return dto.SaveEmergencyContactResponse{}, err
	}

	if err := database.SaveEmergencyContact(emergencyContact); err != nil {
		return dto.SaveEmergencyContactResponse{}, api.ErrorInternalServer(err)
	}

	return dto.SaveEmergencyContactResponse{
		ContactID: emergencyContact.ContactID,
	}, nil
}

// UpdateEmergencyContact a method to update the emergency contact.
func UpdateEmergencyContact(database moira.Database, contactID string, emergencyContactDTO *dto.EmergencyContact) (dto.SaveEmergencyContactResponse, *api.ErrorResponse) {
	if emergencyContactDTO == nil {
		return dto.SaveEmergencyContactResponse{}, nil
	}

	emergencyContact := datatypes.EmergencyContact(*emergencyContactDTO)
	emergencyContact.ContactID = contactID

	if err := database.SaveEmergencyContact(emergencyContact); err != nil {
		return dto.SaveEmergencyContactResponse{}, api.ErrorInternalServer(err)
	}

	return dto.SaveEmergencyContactResponse{
		ContactID: emergencyContact.ContactID,
	}, nil
}

// RemoveEmergencyContact a method to delete the emergency contact.
func RemoveEmergencyContact(database moira.Database, contactID string) *api.ErrorResponse {
	if err := database.RemoveEmergencyContact(contactID); err != nil {
		return api.ErrorInternalServer(err)
	}

	return nil
}