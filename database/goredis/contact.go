package goredis

import (
	"encoding/json"
	"fmt"

	"github.com/go-redis/redis/v8"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database"
	"github.com/moira-alert/moira/database/goredis/reply"
)

// GetContact returns contact data by given id, if no value, return database.ErrNil error
func (connector *DbConnector) GetContact(id string) (moira.ContactData, error) {
	c := *connector.client

	var contact moira.ContactData

	result := c.Get(connector.context, contactKey(id))
	if result.Err() == redis.Nil {
		return contact, database.ErrNil
	}
	contact, err := reply.Contact(*result)
	if err != nil {
		return contact, err
	}
	contact.ID = id
	return contact, nil
}

// GetContacts returns contacts data by given ids, len of contactIDs is equal to len of returned values array.
// If there is no object by current ID, then nil is returned
func (connector *DbConnector) GetContacts(contactIDs []string) ([]*moira.ContactData, error) {
	contacts := make([]*moira.ContactData, 0, len(contactIDs))
	results := make([]*redis.StringCmd, 0, len(contactIDs))

	c := *connector.client
	pipe := c.TxPipeline()

	for _, id := range contactIDs {
		result := pipe.Get(connector.context, contactKey(id))

		results = append(results, result)
	}
	_, err := pipe.Exec(connector.context)
	if err != nil && err != redis.Nil {
		return nil, err
	}

	for i := range results {
		result := results[i]
		if result.Val() != "" {
			contact, err := reply.Contact(*result)
			if err != nil {
				return nil, err
			}
			contacts = append(contacts, &contact)
		} else {
			contacts = append(contacts, nil)
		}
	}

	for i := range contacts {
		if contacts[i] != nil {
			contacts[i].ID = contactIDs[i]
		}
	}
	return contacts, nil
}

// GetAllContacts returns full contact list
func (connector *DbConnector) GetAllContacts() ([]*moira.ContactData, error) {
	c := *connector.client
	keys, err := c.Keys(connector.context, contactKey("*")).Result()
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
func (connector *DbConnector) SaveContact(contact *moira.ContactData) error {
	existing, getContactErr := connector.GetContact(contact.ID)
	if getContactErr != nil && getContactErr != database.ErrNil {
		return getContactErr
	}
	contactString, err := json.Marshal(contact)
	if err != nil {
		return err
	}

	c := *connector.client

	pipe := c.TxPipeline()
	pipe.Set(connector.context, contactKey(contact.ID), contactString, redis.KeepTTL)
	if getContactErr != database.ErrNil && contact.User != existing.User {
		pipe.SRem(connector.context, userContactsKey(existing.User), contact.ID)
	}
	if getContactErr != database.ErrNil && contact.Team != existing.Team {
		pipe.SRem(connector.context, teamContactsKey(existing.Team), contact.ID)
	}
	if contact.User != "" {
		pipe.SAdd(connector.context, userContactsKey(contact.User), contact.ID)
	}
	if contact.Team != "" {
		pipe.SAdd(connector.context, teamContactsKey(contact.Team), contact.ID)
	}
	_, err = pipe.Exec(connector.context)
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
	c := *connector.client

	pipe := c.TxPipeline()
	pipe.Del(connector.context, contactKey(contactID))
	pipe.SRem(connector.context, userContactsKey(existing.User), contactID)
	pipe.SRem(connector.context, teamContactsKey(existing.Team), contactID)
	_, err = pipe.Exec(connector.context)
	if err != nil {
		return fmt.Errorf("failed to EXEC: %s", err.Error())
	}
	return nil
}

// GetUserContactIDs returns contacts ids by given login
func (connector *DbConnector) GetUserContactIDs(login string) ([]string, error) {
	c := *connector.client

	contacts, err := c.SMembers(connector.context, userContactsKey(login)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get contacts for user login %s: %s", login, err.Error())
	}
	return contacts, nil
}

// GetTeamContactIDs returns contacts ids by given team
func (connector *DbConnector) GetTeamContactIDs(login string) ([]string, error) {
	c := *connector.client
	contacts, err := c.SMembers(connector.context, teamContactsKey(login)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get contacts for team login %s: %s", login, err.Error())
	}
	return contacts, nil
}

func contactKey(id string) string {
	return "moira-contact:" + id
}

func userContactsKey(userName string) string {
	return "moira-user-contacts:" + userName
}

func teamContactsKey(userName string) string {
	return "moira-team-contacts:" + userName
}