package redis

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/go-redis/redis/v8"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database"
	"github.com/moira-alert/moira/database/redis/reply"
)

func (connector *DbConnector) GetEmergencyContact(contactID string) (moira.EmergencyContact, error) {
	c := *connector.client
	ctx := connector.context

	cmd := c.Get(ctx, emergencyContactsKey(contactID))

	if errors.Is(cmd.Err(), redis.Nil) {
		return moira.EmergencyContact{}, database.ErrNil
	}

	return reply.EmergencyContact(cmd)
}

func (connector *DbConnector) GetEmergencyContacts() ([]*moira.EmergencyContact, error) {
	c := *connector.client
	ctx := connector.context

	emergencyContactIDs, err := connector.getEmergencyContactIDs()
	if err != nil {
		return nil, fmt.Errorf("failed to get emergency contact IDs: %w", err)
	}

	pipe := c.TxPipeline()
	for _, emergencyContactID := range emergencyContactIDs {
		pipe.Get(ctx, emergencyContactsKey(emergencyContactID))
	}

	cmds, err := pipe.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get emergency contacts by IDs: %w", err)
	}

	emergencyContactCmds := make([]*redis.StringCmd, 0, len(cmds))

	for _, cmd := range cmds {
		emergencyContactCmd, ok := cmd.(*redis.StringCmd)
		if !ok {
			return nil, fmt.Errorf("failed to convert cmd to emergency contact cmd")
		}

		emergencyContactCmds = append(emergencyContactCmds, emergencyContactCmd)
	}

	return reply.EmergencyContacts(emergencyContactCmds)
}

func (connector *DbConnector) getEmergencyContactIDs() ([]string, error) {
	c := *connector.client
	ctx := connector.context

	var emergencyContactIDs []string

	iter := c.Scan(ctx, 0, emergencyContactsKey("*"), 0).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()

		emergencyContactID := strings.TrimPrefix(key, emergencyContactsKey(""))

		emergencyContactIDs = append(emergencyContactIDs, emergencyContactID)
	}

	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan emergency contacts: %w", err)
	}

	return emergencyContactIDs, nil
}

func (connector *DbConnector) GetEmergencyTypeContactIDs(emergencyType moira.EmergencyContactType) ([]string, error) {
	c := *connector.client
	ctx := connector.context

	contactIDs, err := c.SMembers(ctx, emergencyTypeContactsKey(emergencyType)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get emergency type contact IDs '%s': %w", emergencyType, err)
	}

	return contactIDs, nil
}

func (connector *DbConnector) SaveEmergencyContacts(emergencyContacts []moira.EmergencyContact) error {
	c := *connector.client
	ctx := connector.context

	pipe := c.TxPipeline()
	for _, emergencyContact := range emergencyContacts {
		if err := saveEmergencyContactPipe(ctx, pipe, emergencyContact); err != nil {
			return err
		}
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("failed to save emergency contacts: %w", err)
	}

	return nil
}

func (connector *DbConnector) SaveEmergencyContact(emergencyContact moira.EmergencyContact) error {
	c := *connector.client
	ctx := connector.context

	pipe := c.TxPipeline()

	if err := saveEmergencyContactPipe(ctx, pipe, emergencyContact); err != nil {
		return err
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("failed to save emergency contact: %w", err)
	}

	return nil
}

func (connector *DbConnector) RemoveEmergencyContact(contactID string) error {
	c := *connector.client
	ctx := connector.context

	emergencyContact, err := connector.GetEmergencyContact(contactID)
	if err != nil {
		return fmt.Errorf("failed to get emergency contact '%s': %w", contactID, err)
	}

	pipe := c.TxPipeline()

	removeEmergencyContactPipe(ctx, pipe, emergencyContact)

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("failed to remove emergency contact '%s': %w", contactID, err)
	}

	return nil
}

func saveEmergencyContactPipe(ctx context.Context, pipe redis.Pipeliner, emergencyContact moira.EmergencyContact) error {
	emergencyContactBytes, err := reply.GetEmergencyContactBytes(emergencyContact)
	if err != nil {
		return fmt.Errorf("failed to get emergency contact '%s' bytes: %w", emergencyContact.ContactID, err)
	}

	pipe.Set(ctx, emergencyContactsKey(emergencyContact.ContactID), emergencyContactBytes, redis.KeepTTL)

	for _, emergencyType := range emergencyContact.EmergencyTypes {
		pipe.SAdd(ctx, emergencyTypeContactsKey(emergencyType), emergencyContact.ContactID)
	}

	return nil
}

func removeEmergencyContactPipe(ctx context.Context, pipe redis.Pipeliner, emergencyContact moira.EmergencyContact) {
	pipe.Del(ctx, emergencyContactsKey(emergencyContact.ContactID))

	for _, emergencyType := range emergencyContact.EmergencyTypes {
		pipe.SRem(ctx, emergencyTypeContactsKey(emergencyType), emergencyContact.ContactID)
	}
}

func emergencyContactsKey(contactID string) string {
	return "moira-emergency-contacts:" + contactID
}

func emergencyTypeContactsKey(emergencyType moira.EmergencyContactType) string {
	return "moira-emergency-type-contacts:" + string(emergencyType)
}
