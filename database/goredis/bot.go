package goredis

import (
	"fmt"
	"strings"

	"github.com/go-redis/redis/v8"
	"github.com/moira-alert/moira/database"
)

// GetIDByUsername read id of user by messenger username
func (connector *DbConnector) GetIDByUsername(messenger, username string) (string, error) {
	if strings.HasPrefix(username, "#") {
		result := "@" + username[1:]
		return result, nil
	}
	c := *connector.client
	result, err := c.Get(connector.context, usernameKey(messenger, username)).Result()
	if err == redis.Nil {
		return result, database.ErrNil
	}
	return result, err
}

// SetUsernameID store id of username
func (connector *DbConnector) SetUsernameID(messenger, username, id string) error {
	c := *connector.client
	err := c.Set(connector.context, usernameKey(messenger, username), id, redis.KeepTTL).Err()
	return err
}

// RemoveUser removes username from messenger data
func (connector *DbConnector) RemoveUser(messenger, username string) error {
	c := *connector.client
	err := c.Del(connector.context, usernameKey(messenger, username)).Err()
	if err != nil {
		return fmt.Errorf("failed to delete username '%s' from messenger '%s', error: %s", username, messenger, err.Error())
	}
	return nil
}

func usernameKey(messenger, username string) string {
	return fmt.Sprintf("moira-%s-users:%s", messenger, username)
}
