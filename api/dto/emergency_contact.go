package dto

import (
	"errors"
	"net/http"

	"github.com/moira-alert/moira"
)

var (
	emptyEmergencyTypesErr     = errors.New("emergency types can not be empty")
	emptyEmergencyContactIDErr = errors.New("emergency contact id can not be empty")
)

type EmergencyContact struct {
	ContactID      string                       `json:"contact_id" example:"1dd38765-c5be-418d-81fa-7a5f879c2315"`
	EmergencyTypes []moira.EmergencyContactType `json:"emergency_types" example:"notifier_off"`
}

func (*EmergencyContact) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

func (contact *EmergencyContact) Bind(r *http.Request) error {
	if len(contact.EmergencyTypes) == 0 {
		return emptyEmergencyTypesErr
	}

	return nil
}

type EmergencyContactList struct {
	List []EmergencyContact `json:"list"`
}

func (*EmergencyContactList) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

type EmergencyContacts struct {
	Items []EmergencyContact `json:"emergency_contacts"`
}

func (emergencyContacts *EmergencyContacts) Bind(r *http.Request) error {
	for _, emergencyContact := range emergencyContacts.Items {
		if emergencyContact.ContactID == "" {
			return emptyEmergencyContactIDErr
		}

		if len(emergencyContact.EmergencyTypes) == 0 {
			return emptyEmergencyTypesErr
		}
	}

	return nil
}

func FromEmergencyContacts(emergencyContacts []*moira.EmergencyContact) *EmergencyContactList {
	emergencyContactsDTO := &EmergencyContactList{
		List: make([]EmergencyContact, 0, len(emergencyContacts)),
	}

	for _, emergencyContact := range emergencyContacts {
		if emergencyContact != nil {
			emergencyContactsDTO.List = append(emergencyContactsDTO.List, EmergencyContact(*emergencyContact))
		}
	}

	return emergencyContactsDTO
}
