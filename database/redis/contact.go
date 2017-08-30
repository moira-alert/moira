package redis

import (
	"encoding/json"
	"fmt"

	"github.com/garyburd/redigo/redis"
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/database/redis/reply"
)

//GetContact returns contact data by given id, if no value, return database.ErrNil error
func (connector *DbConnector) GetContact(id string) (moira.ContactData, error) {
	c := connector.pool.Get()
	defer c.Close()

	var contact moira.ContactData

	contact, err := reply.Contact(c.Do("GET", moiraContact(id)))
	if err != nil {
		return contact, err
	}
	contact.ID = id
	return contact, nil
}

//GetContacts returns contacts data by given ids, len of contactIDs is equal to len of returned values array.
//If there is no object by current ID, then nil is returned
func (connector *DbConnector) GetContacts(contactIDs []string) ([]*moira.ContactData, error) {
	c := connector.pool.Get()
	defer c.Close()
	c.Send("MULTI")
	for _, id := range contactIDs {
		c.Send("GET", moiraContact(id))
	}

	contacts, err := reply.Contacts(c.Do("EXEC"))
	if err != nil {
		return nil, err
	}
	for i := range contacts {
		if contacts[i] != nil {
			contacts[i].ID = contactIDs[i]
		}
	}
	return contacts, nil
}

//GetAllContacts returns full contact list
func (connector *DbConnector) GetAllContacts() ([]*moira.ContactData, error) {
	c := connector.pool.Get()
	defer c.Close()

	keys, err := redis.Strings(c.Do("KEYS", moiraContact("*")))
	if err != nil {
		return nil, err
	}

	contactIDs := make([]string, 0, len(keys))
	for _, key := range keys {
		key = key[14:]
		contactIDs = append(contactIDs, key)
	}
	return connector.GetContacts(contactIDs)
}

//SaveContact writes contact data and updates user contacts
func (connector *DbConnector) SaveContact(contact *moira.ContactData) error {
	contactString, err := json.Marshal(contact)
	if err != nil {
		return err
	}

	c := connector.pool.Get()
	defer c.Close()

	c.Send("MULTI")
	c.Send("SET", moiraContact(contact.ID), contactString)
	c.Send("SADD", moiraUserContacts(contact.User), contact.ID)
	_, err = c.Do("EXEC")
	if err != nil {
		return fmt.Errorf("Failed to EXEC: %s", err.Error())
	}
	return nil
}

//RemoveContact deletes contact data and contactID from user contacts
func (connector *DbConnector) RemoveContact(contactID string, userLogin string) error {
	c := connector.pool.Get()
	defer c.Close()

	c.Send("MULTI")
	c.Send("DEL", moiraContact(contactID))
	c.Send("SREM", moiraUserContacts(userLogin), contactID)
	_, err := c.Do("EXEC")
	if err != nil {
		return fmt.Errorf("Failed to EXEC: %s", err.Error())
	}
	return nil
}

//WriteContact writes contact data
func (connector *DbConnector) WriteContact(contact *moira.ContactData) error {
	bytes, err := json.Marshal(contact)
	if err != nil {
		return err
	}
	c := connector.pool.Get()
	defer c.Close()
	_, err = c.Do("SET", moiraContact(contact.ID), bytes)
	if err != nil {
		return fmt.Errorf("Failed to set contact data json %s, error: %s", string(bytes), err)
	}
	return nil
}

// GetUserContactIDs returns contacts ids by given login
func (connector *DbConnector) GetUserContactIDs(login string) ([]string, error) {
	c := connector.pool.Get()
	defer c.Close()

	contacts, err := redis.Strings(c.Do("SMEMBERS", moiraUserContacts(login)))
	if err != nil {
		return nil, fmt.Errorf("Failed to get contacts for user login %s: %s", login, err.Error())
	}
	return contacts, nil
}

func moiraContact(id string) string {
	return fmt.Sprintf("moira-contact:%s", id)
}

func moiraUserContacts(userName string) string {
	return fmt.Sprintf("moira-user-contacts:%s", userName)
}
