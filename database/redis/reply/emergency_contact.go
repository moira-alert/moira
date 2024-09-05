package reply

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/go-redis/redis/v8"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database"
)

type emergencyContactStorageElement struct {
	ContactID      string                       `json:"contact_id"`
	EmergencyTypes []moira.EmergencyContactType `json:"emergency_types"`
}

func (se emergencyContactStorageElement) toEmergencyContact() moira.EmergencyContact {
	return moira.EmergencyContact{
		ContactID:      se.ContactID,
		EmergencyTypes: se.EmergencyTypes,
	}
}

func toEmergencyContactStorageElement(emergencyContact moira.EmergencyContact) emergencyContactStorageElement {
	return emergencyContactStorageElement{
		ContactID:      emergencyContact.ContactID,
		EmergencyTypes: emergencyContact.EmergencyTypes,
	}
}

// GetEmergencyContactBytes a method to get bytes of the emergency contact structure stored in Redis.
func GetEmergencyContactBytes(emergencyContact moira.EmergencyContact) ([]byte, error) {
	emergencyContactSE := toEmergencyContactStorageElement(emergencyContact)
	bytes, err := json.Marshal(emergencyContactSE)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal emergency contact storage element: %w", err)
	}

	return bytes, nil
}

func unmarshalEmergencyContact(bytes []byte, err error) (moira.EmergencyContact, error) {
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return moira.EmergencyContact{}, database.ErrNil
		}

		return moira.EmergencyContact{}, fmt.Errorf("failed to read emergency contact: %w", err)
	}

	emergencyContactSE := emergencyContactStorageElement{}
	if err = json.Unmarshal(bytes, &emergencyContactSE); err != nil {
		return moira.EmergencyContact{}, fmt.Errorf("failed to parse emergency contact json %s: %w", string(bytes), err)
	}

	return emergencyContactSE.toEmergencyContact(), nil
}

// EmergencyContacts converts redis DB reply to moira.EmergencyContact objects array.
func EmergencyContacts(rep []*redis.StringCmd) ([]*moira.EmergencyContact, error) {
	if rep == nil {
		return []*moira.EmergencyContact{}, nil
	}

	emergencyContacts := make([]*moira.EmergencyContact, len(rep))

	for i, val := range rep {
		emergencyContact, err := unmarshalEmergencyContact(val.Bytes())
		if err != nil && !errors.Is(err, database.ErrNil) {
			return nil, fmt.Errorf("failed to unmarshal emergency contact: %w", err)
		}

		if errors.Is(err, database.ErrNil) {
			emergencyContacts[i] = nil
		} else {
			emergencyContacts[i] = &emergencyContact
		}
	}

	return emergencyContacts, nil
}

// EmergencyContacts converts redis DB reply to moira.EmergencyContact object.
func EmergencyContact(rep *redis.StringCmd) (moira.EmergencyContact, error) {
	if rep == nil || errors.Is(rep.Err(), redis.Nil) {
		return moira.EmergencyContact{}, database.ErrNil
	}

	emergencyContact, err := unmarshalEmergencyContact(rep.Bytes())
	if err != nil {
		return moira.EmergencyContact{}, fmt.Errorf("failed to unmarshal emergency contact: %w", err)
	}

	return emergencyContact, nil
}
