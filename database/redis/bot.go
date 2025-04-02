package redis

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-redis/redis/v8"
	"github.com/moira-alert/moira/database"
)

// GetChatByUsername read chat of user by messenger username.
func (connector *DbConnector) GetChatByUsername(messenger, username string) (string, error) {
	if strings.HasPrefix(username, "#") {
		result := "@" + username[1:]
		return result, nil
	}

	c := *connector.client
	result, err := c.Get(connector.context, usernameKey(messenger, username)).Result()
	if errors.Is(err, redis.Nil) {
		return result, database.ErrNil
	}

	return result, err
}

// SetUsernameChat store id of username.
func (connector *DbConnector) SetUsernameChat(messenger, username, chatRaw string) error {
	c := *connector.client
	err := c.Set(connector.context, usernameKey(messenger, username), chatRaw, redis.KeepTTL).Err()
	return err
}

// RemoveUser removes username from messenger data.
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
