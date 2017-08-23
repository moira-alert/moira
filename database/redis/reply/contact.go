package reply

import (
	"github.com/moira-alert/moira-alert"
	"encoding/json"
	"github.com/garyburd/redigo/redis"
)

func Contact(rep interface{}, err error) (*moira.ContactData, error) {
	bytes, err := redis.Bytes(rep, err)
	if err != nil {
		return nil, err
	}
	contact := &moira.ContactData{}
	err = json.Unmarshal(bytes, contact)
	if err != nil {
		return nil, err
	}
	return contact, nil
}

func Contacts(rep interface{}, err error) ([]*moira.ContactData, error) {
	values, err := redis.Values(rep, err)
	if err != nil {
		return nil, err
	}
	contacts := make([]*moira.ContactData, len(values))
	for i, kk := range values {
		contact, err2 := Contact(kk, err)
		if err2 != nil {
			return nil, err2
		}
		contacts[i] = contact
	}
	return contacts, nil
}
