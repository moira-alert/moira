package redis

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/go-redis/redis/v8"
	"github.com/moira-alert/moira/database"
	"github.com/moira-alert/moira/database/redis/reply"
	"github.com/moira-alert/moira/datatypes"
)

// GetEmergencyContact method to retrieve an emergency contact from the database.
func (connector *DbConnector) GetEmergencyContact(contactID string) (datatypes.EmergencyContact, error) {
	c := *connector.client
	ctx := connector.context

	cmd := c.Get(ctx, emergencyContactsKey(contactID))

	if errors.Is(cmd.Err(), redis.Nil) {
		return datatypes.EmergencyContact{}, database.ErrNil
	}

	return reply.EmergencyContact(cmd)
}

// GetEmergencyContacts method to retrieve all emergency contacts from the database.
func (connector *DbConnector) GetEmergencyContacts() ([]*datatypes.EmergencyContact, error) {
	emergencyContactIDs, err := connector.getEmergencyContactIDs()
	if err != nil {
		return nil, fmt.Errorf("failed to get emergency contact IDs: %w", err)
	}

	return connector.GetEmergencyContactsByIDs(emergencyContactIDs)
}

// GetEmergencyContactsByIDs method to retrieve all emergency contacts from the database by their identifiers.
func (connector *DbConnector) GetEmergencyContactsByIDs(contactIDs []string) ([]*datatypes.EmergencyContact, error) {
	c := *connector.client
	ctx := connector.context

	pipe := c.TxPipeline()
	for _, contactID := range contactIDs {
		pipe.Get(ctx, emergencyContactsKey(contactID))
	}

	cmds, err := pipe.Exec(ctx)
	if err != nil && !errors.Is(err, redis.Nil) {
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

// GetHeartbeatTypeContactIDs a method for obtaining contact IDs by specific emergency type.
func (connector *DbConnector) GetHeartbeatTypeContactIDs(heartbeatType datatypes.HeartbeatType) ([]string, error) {
	c := *connector.client
	ctx := connector.context

	contactIDs, err := c.SMembers(ctx, heartbeatTypeContactsKey(heartbeatType)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get heartbeat type contact IDs '%s': %w", heartbeatType, err)
	}

	return contactIDs, nil
}

func (connector *DbConnector) saveEmergencyContacts(emergencyContacts []datatypes.EmergencyContact) error {
	c := *connector.client
	ctx := connector.context

	pipe := c.TxPipeline()
	for _, emergencyContact := range emergencyContacts {
		if err := addSaveEmergencyContactToPipe(ctx, pipe, emergencyContact); err != nil {
			return fmt.Errorf("failed to add save emergency contact '%s' to pipe: %w", emergencyContact.ContactID, err)
		}
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("failed to save emergency contacts: %w", err)
	}

	return nil
}

// SaveEmergencyContact a method for saving emergency contact.
func (connector *DbConnector) SaveEmergencyContact(emergencyContact datatypes.EmergencyContact) error {
	c := *connector.client
	ctx := connector.context

	pipe := c.TxPipeline()

	if err := addSaveEmergencyContactToPipe(ctx, pipe, emergencyContact); err != nil {
		return err
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("failed to save emergency contact '%s': %w", emergencyContact.ContactID, err)
	}

	return nil
}

// RemoveEmergencyContact method for removing emergency contact.
func (connector *DbConnector) RemoveEmergencyContact(contactID string) error {
	c := *connector.client
	ctx := connector.context

	emergencyContact, err := connector.GetEmergencyContact(contactID)
	if err != nil {
		return fmt.Errorf("failed to get emergency contact '%s': %w", contactID, err)
	}

	pipe := c.TxPipeline()

	addRemoveEmergencyContactToPipe(ctx, pipe, emergencyContact)

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("failed to remove emergency contact '%s': %w", contactID, err)
	}

	return nil
}

func addSaveEmergencyContactToPipe(ctx context.Context, pipe redis.Pipeliner, emergencyContact datatypes.EmergencyContact) error {
	emergencyContactBytes, err := reply.GetEmergencyContactBytes(emergencyContact)
	if err != nil {
		return fmt.Errorf("failed to get emergency contact '%s' bytes: %w", emergencyContact.ContactID, err)
	}

	pipe.Set(ctx, emergencyContactsKey(emergencyContact.ContactID), emergencyContactBytes, redis.KeepTTL)

	for _, heartbeatType := range emergencyContact.HeartbeatTypes {
		pipe.SAdd(ctx, heartbeatTypeContactsKey(heartbeatType), emergencyContact.ContactID)
	}

	return nil
}

func addRemoveEmergencyContactToPipe(ctx context.Context, pipe redis.Pipeliner, emergencyContact datatypes.EmergencyContact) {
	pipe.Del(ctx, emergencyContactsKey(emergencyContact.ContactID))

	for _, heartbeatType := range emergencyContact.HeartbeatTypes {
		pipe.SRem(ctx, heartbeatTypeContactsKey(heartbeatType), emergencyContact.ContactID)
	}
}

func emergencyContactsKey(contactID string) string {
	return "moira-emergency-contacts:" + contactID
}

func heartbeatTypeContactsKey(heartbeatType datatypes.HeartbeatType) string {
	return "moira-heartbeat-type-contacts:" + string(heartbeatType)
}