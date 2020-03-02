package redis

import (
	"encoding/json"
	"fmt"

	moira2 "github.com/moira-alert/moira/internal/moira"

	"github.com/gomodule/redigo/redis"
	"github.com/moira-alert/moira/internal/database"
	"github.com/moira-alert/moira/internal/database/redis/reply"
)

// GetContact returns contact data by given id, if no value, return database.ErrNil error
func (connector *DbConnector) GetContact(id string) (moira2.ContactData, error) {
	c := connector.pool.Get()
	defer c.Close()

	var contact moira2.ContactData

	contact, err := reply.Contact(c.Do("GET", contactKey(id)))
	if err != nil {
		return contact, err
	}
	contact.ID = id
	return contact, nil
}

// GetContacts returns contacts data by given ids, len of contactIDs is equal to len of returned values array.
// If there is no object by current ID, then nil is returned
func (connector *DbConnector) GetContacts(contactIDs []string) ([]*moira2.ContactData, error) {
	c := connector.pool.Get()
	defer c.Close()
	c.Send("MULTI")
	for _, id := range contactIDs {
		c.Send("GET", contactKey(id))
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

// GetAllContacts returns full contact list
func (connector *DbConnector) GetAllContacts() ([]*moira2.ContactData, error) {
	c := connector.pool.Get()
	defer c.Close()

	keys, err := redis.Strings(c.Do("KEYS", contactKey("*")))
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

// SaveContact writes contact data and updates user contacts
func (connector *DbConnector) SaveContact(contact *moira2.ContactData) error {
	existing, getContactErr := connector.GetContact(contact.ID)
	if getContactErr != nil && getContactErr != database.ErrNil {
		return getContactErr
	}
	contactString, err := json.Marshal(contact)
	if err != nil {
		return err
	}

	c := connector.pool.Get()
	defer c.Close()

	c.Send("MULTI")
	c.Send("SET", contactKey(contact.ID), contactString)
	if getContactErr != database.ErrNil && contact.User != existing.User {
		c.Send("SREM", userContactsKey(existing.User), contact.ID)
	}
	c.Send("SADD", userContactsKey(contact.User), contact.ID)
	_, err = c.Do("EXEC")
	if err != nil {
		return fmt.Errorf("failed to EXEC: %s", err.Error())
	}
	return nil
}

// RemoveContact deletes contact data and contactID from user contacts
func (connector *DbConnector) RemoveContact(contactID string) error {
	existing, err := connector.GetContact(contactID)
	if err != nil && err != database.ErrNil {
		return err
	}
	c := connector.pool.Get()
	defer c.Close()

	c.Send("MULTI")
	c.Send("DEL", contactKey(contactID))
	c.Send("SREM", userContactsKey(existing.User), contactID)
	_, err = c.Do("EXEC")
	if err != nil {
		return fmt.Errorf("failed to EXEC: %s", err.Error())
	}
	return nil
}

// GetUserContactIDs returns contacts ids by given login
func (connector *DbConnector) GetUserContactIDs(login string) ([]string, error) {
	c := connector.pool.Get()
	defer c.Close()

	contacts, err := redis.Strings(c.Do("SMEMBERS", userContactsKey(login)))
	if err != nil {
		return nil, fmt.Errorf("failed to get contacts for user login %s: %s", login, err.Error())
	}
	return contacts, nil
}

func contactKey(id string) string {
	return fmt.Sprintf("moira-contact:%s", id)
}

func userContactsKey(userName string) string {
	return fmt.Sprintf("moira-user-contacts:%s", userName)
}
