package dto

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/moira-alert/moira/api/middleware"
	"github.com/moira-alert/moira/datatypes"
)

// ErrEmptyHeartbeatTypes means that the user has not specified any heartbeat types.
var ErrEmptyHeartbeatTypes = errors.New("heartbeat types can not be empty")

// EmergencyContact is the DTO structure for contacts to which notifications will go in the event of special internal Moira problems.
type EmergencyContact struct {
	ContactID      string                    `json:"contact_id" example:"1dd38765-c5be-418d-81fa-7a5f879c2315"`
	HeartbeatTypes []datatypes.HeartbeatType `json:"heartbeat_types" example:"notifier_off"`
}

// Render is a function that implements chi Renderer interface for EmergencyContact.
func (*EmergencyContact) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

// Bind is a method that implements Binder interface from chi and checks that validity of data in request.
func (emergencyContact *EmergencyContact) Bind(r *http.Request) error {
	if len(emergencyContact.HeartbeatTypes) == 0 {
		return ErrEmptyHeartbeatTypes
	}

	auth := middleware.GetAuth(r)
	userLogin := middleware.GetLogin(r)
	isAdmin := auth.IsAdmin(userLogin)

	for _, emergencyType := range emergencyContact.HeartbeatTypes {
		if !emergencyType.IsValid() {
			return fmt.Errorf("'%s' heartbeat type doesn't exist", emergencyType)
		}

		if _, ok := auth.AllowedEmergencyContactTypes[emergencyType]; !ok && !isAdmin {
			return fmt.Errorf("'%s' heartbeat type is not allowed", emergencyType)
		}
	}

	return nil
}

// EmergencyContactList is the DTO structure for list of contacts to which notifications will go in the event of special internal Moira problems.
type EmergencyContactList struct {
	List []EmergencyContact `json:"list"`
}

// Render is a function that implements chi Renderer interface for EmergencyContactList.
func (*EmergencyContactList) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

// FromEmergencyContacts a method that converts emergency contacts to dto emergency ccontact list.
func FromEmergencyContacts(emergencyContacts []*datatypes.EmergencyContact) *EmergencyContactList {
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

// SaveEmergencyContactResponse is the DTO structure which is returned in the methods of saving contact.
type SaveEmergencyContactResponse struct {
	ContactID string `json:"contact_id" example:"1dd38765-c5be-418d-81fa-7a5f879c2315"`
}

// Render is a function that implements chi Renderer interface for SaveEmergencyContactResponse.
func (SaveEmergencyContactResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
