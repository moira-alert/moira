package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/go-redis/redis/v8"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database"
	"github.com/moira-alert/moira/database/redis/reply"
)

// GetContact returns contact data by given id, if no value, return database.ErrNil error.
func (connector *DbConnector) GetContact(id string) (moira.ContactData, error) {
	c := *connector.client
	ctx := connector.context

	var contact moira.ContactData

	result := c.Get(ctx, contactKey(id))
	if errors.Is(result.Err(), redis.Nil) {
		return contact, database.ErrNil
	}

	contact, err := reply.Contact(result)
	if err != nil {
		return contact, fmt.Errorf("failed to deserialize contact '%s': %w", id, err)
	}

	contact.ID = id

	return contact, nil
}

// GetContacts returns contacts data by given ids, len of contactIDs is equal to len of returned values array.
// If there is no object by current ID, then nil is returned.
func (connector *DbConnector) GetContacts(contactIDs []string) ([]*moira.ContactData, error) {
	results := make([]*redis.StringCmd, 0, len(contactIDs))

	c := *connector.client
	ctx := connector.context

	pipe := c.TxPipeline()
	for _, id := range contactIDs {
		result := pipe.Get(ctx, contactKey(id))
		results = append(results, result)
	}

	_, err := pipe.Exec(ctx)
	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, fmt.Errorf("failed to get contacts by id: %w", err)
	}

	contacts, err := reply.Contacts(results)
	if err != nil {
		return nil, fmt.Errorf("failed to reply contacts: %w", err)
	}

	for i := range contacts {
		if contacts[i] != nil {
			contacts[i].ID = contactIDs[i]
		}
	}

	return contacts, nil
}

func getContactsKeysOnRedisNode(ctx context.Context, client redis.UniversalClient) ([]string, error) {
	var cursor uint64
	var keys []string
	const scanCount = 10000

	for {
		var keysResult []string
		var err error
		keysResult, cursor, err = client.Scan(ctx, cursor, contactKey("*"), scanCount).Result()
		if err != nil {
			return nil, err
		}

		keys = append(keys, keysResult...)

		if cursor == 0 {
			break
		}
	}

	return keys, nil
}

// GetAllContacts returns full contact list.
func (connector *DbConnector) GetAllContacts() ([]*moira.ContactData, error) {
	var keys []string

	err := connector.callFunc(func(connector *DbConnector, client redis.UniversalClient) error {
		keysResult, err := getContactsKeysOnRedisNode(connector.context, client)
		if err != nil {
			return err
		}

		keys = append(keys, keysResult...)
		return nil
	})
	if err != nil {
		return nil, err
	}

	contactIDs := make([]string, 0, len(keys))
	for _, key := range keys {
		contactIDs = append(contactIDs, strings.TrimPrefix(key, contactKey("")))
	}

	return connector.GetContacts(contactIDs)
}

// SaveContact writes contact data and updates user contacts.
func (connector *DbConnector) SaveContact(contact *moira.ContactData) error {
	existing, getContactErr := connector.GetContact(contact.ID)
	if getContactErr != nil && !errors.Is(getContactErr, database.ErrNil) {
		return fmt.Errorf("failed to get contact '%s': %w", contact.ID, getContactErr)
	}

	contactStr, err := json.Marshal(contact)
	if err != nil {
		return fmt.Errorf("failed to marshal contact '%s': %w", contact.ID, err)
	}

	c := *connector.client
	ctx := connector.context

	pipe := c.TxPipeline()
	pipe.Set(ctx, contactKey(contact.ID), contactStr, redis.KeepTTL)
	if !errors.Is(getContactErr, database.ErrNil) && contact.User != existing.User {
		pipe.SRem(ctx, userContactsKey(existing.User), contact.ID)
	}

	if !errors.Is(getContactErr, database.ErrNil) && contact.Team != existing.Team {
		pipe.SRem(ctx, teamContactsKey(existing.Team), contact.ID)
	}

	if contact.User != "" {
		pipe.SAdd(ctx, userContactsKey(contact.User), contact.ID)
	}

	if contact.Team != "" {
		pipe.SAdd(ctx, teamContactsKey(contact.Team), contact.ID)
	}

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to save contact '%s': %w", contact.ID, err)
	}

	return nil
}

// RemoveContact deletes contact data and contactID from user contacts.
func (connector *DbConnector) RemoveContact(contactID string) error {
	existing, err := connector.GetContact(contactID)
	if err != nil && !errors.Is(err, database.ErrNil) {
		return fmt.Errorf("failed to get contact '%s': %w", contactID, err)
	}

	emergencyContact, getEmergencyContactErr := connector.GetEmergencyContact(contactID)
	if getEmergencyContactErr != nil && !errors.Is(getEmergencyContactErr, database.ErrNil) {
		return fmt.Errorf("failed to get emergency contact '%s': %w", contactID, err)
	}

	c := *connector.client
	ctx := connector.context

	pipe := c.TxPipeline()
	pipe.Del(ctx, contactKey(contactID))
	pipe.SRem(ctx, userContactsKey(existing.User), contactID)
	pipe.SRem(ctx, teamContactsKey(existing.Team), contactID)

	if !errors.Is(getEmergencyContactErr, database.ErrNil) {
		addRemoveEmergencyContactToPipe(ctx, pipe, emergencyContact)
	}

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to remove contact '%s': %w", contactID, err)
	}

	return nil
}

// GetUserContactIDs returns contacts ids by given login.
func (connector *DbConnector) GetUserContactIDs(login string) ([]string, error) {
	c := *connector.client
	ctx := connector.context

	contactIDs, err := c.SMembers(ctx, userContactsKey(login)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get contact IDs for user login '%s': %w", login, err)
	}

	return contactIDs, nil
}

// GetTeamContactIDs returns contacts ids by given team.
func (connector *DbConnector) GetTeamContactIDs(login string) ([]string, error) {
	c := *connector.client
	ctx := connector.context

	contactIDs, err := c.SMembers(ctx, teamContactsKey(login)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get contact IDs for team login '%s': %w", login, err)
	}

	return contactIDs, nil
}

func contactKey(id string) string {
	return "moira-contact:" + id
}

func userContactsKey(userName string) string {
	return "moira-user-contacts:" + userName
}

func teamContactsKey(teamName string) string {
	return "moira-team-contacts:" + teamName
}
