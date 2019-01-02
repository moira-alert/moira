package redis

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	"strings"

	"github.com/moira-alert/moira/database"
)

// GetIDByUsername read ID of user by messenger username
func (connector *DbConnector) GetIDByUsername(messenger, username string) (string, error) {
	if strings.HasPrefix(username, "#") {
		result := "@" + username[1:]
		return result, nil
	}
	c := connector.pool.Get()
	defer c.Close()
	result, err := redis.String(c.Do("GET", usernameKey(messenger, username)))
	if err == redis.ErrNil {
		return result, database.ErrNil
	}
	return result, err
}

// SetUsernameID store id of username
func (connector *DbConnector) SetUsernameID(messenger, username, id string) error {
	c := connector.pool.Get()
	defer c.Close()
	_, err := c.Do("SET", usernameKey(messenger, username), id)
	return err
}

// RemoveUser removes username from messenger data
func (connector *DbConnector) RemoveUser(messenger, username string) error {
	c := connector.pool.Get()
	defer c.Close()
	_, err := c.Do("DEL", usernameKey(messenger, username))
	if err != nil {
		return fmt.Errorf("Failed to delete username '%s' from messenger '%s', error: %s", username, messenger, err.Error())
	}
	return nil
}

func usernameKey(messenger, username string) string {
	return fmt.Sprintf("moira-%s-users:%s", messenger, username)
}
