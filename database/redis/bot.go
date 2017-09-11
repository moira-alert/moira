package redis

import (
	"fmt"
	"os"
	"strings"

	"github.com/garyburd/redigo/redis"
	"github.com/moira-alert/moira/database"
)

const (
	botUsername  = "moira-bot-host"
	deregistered = "deregistered"
)

var messengers = make(map[string]bool)

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

// RegisterBotIfAlreadyNot creates registration of bot instance in redis
func (connector *DbConnector) RegisterBotIfAlreadyNot(messenger string) bool {
	host, _ := os.Hostname()
	redisKey := usernameKey(messenger, botUsername)
	c := connector.pool.Get()
	defer c.Close()

	c.Send("WATCH", redisKey)

	status, err := connector.GetIDByUsername(messenger, botUsername)
	if err != nil && err != database.ErrNil {
		connector.logger.Info(err)
	}
	if status == "" || status == host || status == deregistered {
		c.Send("MULTI")
		c.Send("SET", redisKey, host)
		_, err := c.Do("EXEC")
		if err != nil {
			connector.logger.Info(err)
			return false
		}
		messengers[messenger] = true
		return true
	}

	return false
}

// DeregisterBots cancels registration for all registered messengers
func (connector *DbConnector) DeregisterBots() {
	for messenger, ok := range messengers {
		if ok {
			connector.DeregisterBot(messenger)
		}
	}
}

// DeregisterBot removes registration of bot instance in redis
func (connector *DbConnector) DeregisterBot(messenger string) error {
	status, _ := connector.GetIDByUsername(messenger, botUsername)
	host, _ := os.Hostname()
	if status == host {
		connector.logger.Debugf("Bot for %s on host %s exists. Removing registration.", messenger, host)
		delete(messengers, messenger)
		return connector.SetUsernameID(messenger, botUsername, deregistered)
	}

	connector.logger.Debugf("Notifier on host %s did't exist. Removing skipped.", host)
	return nil
}

func usernameKey(messenger, username string) string {
	return fmt.Sprintf("moira-%s-users:%s", messenger, username)
}
