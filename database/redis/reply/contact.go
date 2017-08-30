package reply

import (
	"encoding/json"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/database"
)

func Contact(rep interface{}, err error) (moira.ContactData, error) {
	contact := moira.ContactData{}
	bytes, err := redis.Bytes(rep, err)
	if err != nil {
		if err == redis.ErrNil {
			return contact, database.ErrNil
		}
		return contact, fmt.Errorf("Failed to read contact: %s", err.Error())
	}
	err = json.Unmarshal(bytes, &contact)
	if err != nil {
		return contact, fmt.Errorf("Failed to parse contact json %s: %s", string(bytes), err.Error())
	}
	return contact, nil
}

func Contacts(rep interface{}, err error) ([]*moira.ContactData, error) {
	values, err := redis.Values(rep, err)
	if err != nil {
		if err == redis.ErrNil {
			return make([]*moira.ContactData, 0), nil
		}
		return nil, fmt.Errorf("Failed to read contacts: %s", err.Error())
	}
	contacts := make([]*moira.ContactData, len(values))
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
