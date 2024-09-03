package controller

import (
	"errors"
	"fmt"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/dto"
	moiradb "github.com/moira-alert/moira/database"
)

func GetEmergencyContacts(database moira.Database) (*dto.EmergencyContactList, *api.ErrorResponse) {
	emergencyContacts, err := database.GetEmergencyContacts()
	if err != nil {
		return nil, api.ErrorInternalServer(err)
	}

	return dto.FromEmergencyContacts(emergencyContacts), nil
}

func GetEmergencyContact(database moira.Database, contactID string) (*dto.EmergencyContact, *api.ErrorResponse) {
	emergencyContact, err := database.GetEmergencyContact(contactID)
	if err != nil {
		if errors.Is(err, moiradb.ErrNil) {
			return nil, api.ErrorNotFound(fmt.Sprintf("contact with ID '%s' does not exists", contactID))
		}

		return nil, api.ErrorInternalServer(err)
	}

	emergencyContactDTO := dto.EmergencyContact(emergencyContact)

	return &emergencyContactDTO, nil
}

func verifyEmergencyContactAccess(
	database moira.Database,
	auth *api.Authorization,
	emergencyContact moira.EmergencyContact,
	userLogin string,
) *api.ErrorResponse {
	contact, err := database.GetContact(emergencyContact.ContactID)
	if err != nil {
		api.ErrorInternalServer(err)
	}

	// Only admins are allowed to create an emergency contacts for other users
	if !auth.IsAdmin(userLogin) && contact.User != "" && contact.User != userLogin {
		return api.ErrorInvalidRequest(fmt.Errorf("cannot create an emergency contact using someone else's contact_id '%s'", emergencyContact.ContactID))
	}

	return nil
}

func CreateEmergencyContacts(
	database moira.Database,
	auth *api.Authorization,
	emergencyContactsDTO *dto.EmergencyContacts,
	userLogin string,
) *api.ErrorResponse {
	if emergencyContactsDTO == nil {
		return nil
	}

	emergencyContacts := make([]moira.EmergencyContact, 0, len(emergencyContactsDTO.Items))

	for _, emergencyContactDTO := range emergencyContactsDTO.Items {
		emergencyContact := moira.EmergencyContact(emergencyContactDTO)

		if err := verifyEmergencyContactAccess(database, auth, emergencyContact, userLogin); err != nil {
			return err
		}

		emergencyContacts = append(emergencyContacts, emergencyContact)
	}

	if err := database.SaveEmergencyContacts(emergencyContacts); err != nil {
		return api.ErrorInternalServer(err)
	}

	return nil
}

func CreateEmergencyContact(
	database moira.Database,
	auth *api.Authorization,
	emergencyContactDTO *dto.EmergencyContact,
	userLogin string,
) *api.ErrorResponse {
	if emergencyContactDTO == nil {
		return nil
	}

	emergencyContact := moira.EmergencyContact(*emergencyContactDTO)

	if err := verifyEmergencyContactAccess(database, auth, emergencyContact, userLogin); err != nil {
		return err
	}

	if err := database.SaveEmergencyContact(emergencyContact); err != nil {
		return api.ErrorInternalServer(err)
	}

	return nil
}

func UpdateEmergencyContact(database moira.Database, contactID string, emergencyContactDTO *dto.EmergencyContact) (*dto.EmergencyContact, *api.ErrorResponse) {
	if emergencyContactDTO == nil {
		return nil, nil
	}

	emergencyContact := moira.EmergencyContact(*emergencyContactDTO)
	emergencyContact.ContactID = contactID

	if err := database.SaveEmergencyContact(emergencyContact); err != nil {
		return nil, api.ErrorInternalServer(err)
	}

	updatedEmergencyContactDTO := dto.EmergencyContact(emergencyContact)

	return &updatedEmergencyContactDTO, nil
}

func RemoveEmergencyContact(database moira.Database, contactID string) *api.ErrorResponse {
	if err := database.RemoveEmergencyContact(contactID); err != nil {
		return api.ErrorInternalServer(err)
	}

	return nil
}
