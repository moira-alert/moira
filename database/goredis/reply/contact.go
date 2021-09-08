package reply

import (
	"encoding/json"
	"fmt"

	"github.com/moira-alert/moira/database"

	"github.com/go-redis/redis/v8"
	"github.com/moira-alert/moira"
)

func unmarshalContact(bytes []byte, err error) (moira.ContactData, error) {
	contact := moira.ContactData{}
	if err != nil {
		if err == redis.Nil {
			return contact, database.ErrNil
		}
		return contact, fmt.Errorf("failed to read contact: %s", err.Error())
	}

	err = json.Unmarshal(bytes, &contact)
	if err != nil {
		return contact, fmt.Errorf("failed to parse contact json %s: %s", string(bytes), err.Error())
	}

	return contact, nil
}

// Contact converts redis DB reply to moira.ContactData object
func Contact(rep *redis.StringCmd) (moira.ContactData, error) {
	return unmarshalContact(rep.Bytes())
}

// Contacts converts redis DB reply to moira.ContactData objects array
func Contacts(rep []*redis.StringCmd) ([]*moira.ContactData, error) {
	contacts := make([]*moira.ContactData, len(rep))
	for i, value := range rep {
		contact, err := unmarshalContact(value.Bytes())
		if err != nil && err != database.ErrNil {
			return nil, err
		} else if err == database.ErrNil {
			contacts[i] = nil
		} else {
			contacts[i] = &contact
		}
	}
	return contacts, nil
}
