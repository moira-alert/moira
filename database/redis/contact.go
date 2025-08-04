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

	var contact moira.ContactData

	result := c.Get(connector.context, contactKey(id))
	if errors.Is(result.Err(), redis.Nil) {
		return contact, database.ErrNil
	}

	contact, err := reply.Contact(result)
	if err != nil {
		return contact, err
	}

	contact.ID = id

	return contact, nil
}

// GetContacts returns contacts data by given ids, len of contactIDs is equal to len of returned values array.
// If there is no object by current ID, then nil is returned.
func (connector *DbConnector) GetContacts(contactIDs []string) ([]*moira.ContactData, error) {
	results := make([]*redis.StringCmd, 0, len(contactIDs))

	c := *connector.client

	pipe := c.TxPipeline()
	for _, id := range contactIDs {
		result := pipe.Get(connector.context, contactKey(id))
		results = append(results, result)
	}

	_, err := pipe.Exec(connector.context)
	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, err
	}

	contacts, err := reply.Contacts(results)
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
		return getContactErr
	}

	contactString, err := json.Marshal(contact)
	if err != nil {
		return err
	}

	c := *connector.client

	pipe := c.TxPipeline()
	pipe.Set(connector.context, contactKey(contact.ID), contactString, redis.KeepTTL)
	pipe.Del(connector.context, contactScoreKey(contact.ID))

	if !errors.Is(getContactErr, database.ErrNil) && contact.User != existing.User {
		pipe.SRem(connector.context, userContactsKey(existing.User), contact.ID)
	}

	if !errors.Is(getContactErr, database.ErrNil) && contact.Team != existing.Team {
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

// RemoveContact deletes contact data and contactID from user contacts.
func (connector *DbConnector) RemoveContact(contactID string) error {
	existing, err := connector.GetContact(contactID)
	if err != nil && !errors.Is(err, database.ErrNil) {
		return err
	}

	c := *connector.client

	pipe := c.TxPipeline()
	pipe.Del(connector.context, contactKey(contactID))
	pipe.Del(connector.context, contactScoreKey(contactID))
	pipe.SRem(connector.context, userContactsKey(existing.User), contactID)
	pipe.SRem(connector.context, teamContactsKey(existing.Team), contactID)

	_, err = pipe.Exec(connector.context)
	if err != nil {
		return fmt.Errorf("failed to EXEC: %s", err.Error())
	}

	return nil
}

// GetUserContactIDs returns contacts ids by given login.
func (connector *DbConnector) GetUserContactIDs(login string) ([]string, error) {
	c := *connector.client

	contacts, err := c.SMembers(connector.context, userContactsKey(login)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get contacts for user login %s: %s", login, err.Error())
	}

	return contacts, nil
}

// GetTeamContactIDs returns contacts ids by given team.
func (connector *DbConnector) GetTeamContactIDs(login string) ([]string, error) {
	c := *connector.client

	contacts, err := c.SMembers(connector.context, teamContactsKey(login)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get contacts for team login %s: %s", login, err.Error())
	}

	return contacts, nil
}

// UpdateContactScores updates the scores of contacts in the database based on the provided IDs and updater function.
func (connector *DbConnector) UpdateContactScores(contactIDs []string, updater func(moira.ContactScore) moira.ContactScore) error {
	c := *connector.client
	ctx := connector.context
	contactScoresIDs := moira.Map(contactIDs, func(contactID string) string { return contactScoreKey(contactID) })

	return c.Watch(ctx, func(tx *redis.Tx) error {
		pipe := tx.Pipeline()
		cmds := make([]*redis.StringCmd, len(contactScoresIDs))

		for i, key := range contactScoresIDs {
			cmds[i] = pipe.Get(ctx, key)
		}

		_, err := pipe.Exec(ctx)
		if err != nil && !errors.Is(err, redis.Nil) {
			return err
		}

		_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			for i, cmd := range cmds {
				data, err := cmd.Result()
				if err != nil && !errors.Is(err, redis.Nil) {
					return err
				}

				contactScore := moira.ContactScore{ContactID: contactIDs[i]}
				if data != "" {
					if err := json.Unmarshal([]byte(data), &contactScore); err != nil {
						return err
					}
				}

				updatedScore := updater(contactScore)

				updatedData, err := json.Marshal(updatedScore)
				if err != nil {
					return err
				}

				pipe.Set(ctx, contactScoresIDs[i], updatedData, 0)
			}

			return nil
		})

		return err
	}, contactScoresIDs...)
}

// GetContactsScore returns contacts scores as map[contactID]ContactScore.
func (connector *DbConnector) GetContactsScore(contactIDs []string) (map[string]*moira.ContactScore, error) {
	c := *connector.client

	contactScores := make(map[string]*moira.ContactScore, len(contactIDs))

	for _, contactID := range contactIDs {
		var contactScore moira.ContactScore

		result := c.Get(connector.context, contactScoreKey(contactID))

		err := result.Err()
		if errors.Is(err, redis.Nil) {
			continue
		}

		if err != nil {
			return nil, err
		}

		err = json.Unmarshal([]byte(result.Val()), &contactScore)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal contact score: %s", err.Error())
		}

		contactScores[contactID] = &contactScore
	}

	return contactScores, nil
}

// GetContactScore returns contact score by given contact id.
func (connector *DbConnector) GetContactScore(contactID string) (*moira.ContactScore, error) {
	scores, err := connector.GetContactsScore([]string{contactID})
	if err != nil {
		return nil, err
	}

	return scores[contactID], nil
}

func contactKey(id string) string {
	return "moira-contact:" + id
}

func contactScoreKey(id string) string {
	return "moira-contact-score:" + id
}

func userContactsKey(userName string) string {
	return "moira-user-contacts:" + userName
}

func teamContactsKey(teamName string) string {
	return "moira-team-contacts:" + teamName
}
