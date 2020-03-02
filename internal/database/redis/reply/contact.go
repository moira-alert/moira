package reply

import (
	"encoding/json"
	"fmt"

	moira2 "github.com/moira-alert/moira/internal/moira"

	"github.com/gomodule/redigo/redis"
	"github.com/moira-alert/moira/internal/database"
)

// Contact converts redis DB reply to moira.ContactData object
func Contact(rep interface{}, err error) (moira2.ContactData, error) {
	contact := moira2.ContactData{}
	bytes, err := redis.Bytes(rep, err)
	if err != nil {
		if err == redis.ErrNil {
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

// Contacts converts redis DB reply to moira.ContactData objects array
func Contacts(rep interface{}, err error) ([]*moira2.ContactData, error) {
	values, err := redis.Values(rep, err)
	if err != nil {
		if err == redis.ErrNil {
			return make([]*moira2.ContactData, 0), nil
		}
		return nil, fmt.Errorf("failed to read contacts: %s", err.Error())
	}
	contacts := make([]*moira2.ContactData, len(values))
	for i, value := range values {
		contact, err2 := Contact(value, err)
		if err2 != nil && err2 != database.ErrNil {
			return nil, err2
		} else if err2 == database.ErrNil {
			contacts[i] = nil
		} else {
			contacts[i] = &contact
		}
	}
	return contacts, nil
}
