package dto

import (
	"errors"
	"net/http"

	"github.com/moira-alert/moira"
)

var (
	errEmptyEmergencyTypes     = errors.New("emergency types can not be empty")
)

type EmergencyContact struct {
	ContactID      string                       `json:"contact_id" example:"1dd38765-c5be-418d-81fa-7a5f879c2315"`
	EmergencyTypes []moira.EmergencyContactType `json:"emergency_types" example:"notifier_off"`
}

func (*EmergencyContact) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

func (emergencyContact *EmergencyContact) Bind(r *http.Request) error {
	if len(emergencyContact.EmergencyTypes) == 0 {
		return errEmptyEmergencyTypes
	}

	return nil
}

type EmergencyContactList struct {
	List []EmergencyContact `json:"list"`
}

func (*EmergencyContactList) Render(w http.ResponseWriter, r *http.Request) error {
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

type SaveEmergencyContactResponse struct {
	ContactID string `json:"contact_id" example:"1dd38765-c5be-418d-81fa-7a5f879c2315"`
}

func (SaveEmergencyContactResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
