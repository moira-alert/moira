package redis

import (
	"fmt"
	"strings"
	"time"

	"github.com/garyburd/redigo/redis"
	"gopkg.in/redsync.v1"

	"github.com/moira-alert/moira/database"
	"github.com/patrickmn/go-cache"
)

const (
	botUsername = "moira-bot-host"
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

// RegisterBotIfAlreadyNot creates registration of bot instance in redis
func (connector *DbConnector) RegisterBotIfAlreadyNot(messenger string, ttl time.Duration) bool {
	mutex := connector.sync.NewMutex(usernameKey(messenger, botUsername), redsync.SetExpiry(ttl), redsync.SetTries(1))
	if err := mutex.Lock(); err != nil {
		return false
	}
	connector.messengersCache.Set(messenger, mutex, cache.NoExpiration)
	return true
}

// RenewBotRegistration extends bot lock registrations for given ttl
func (connector *DbConnector) RenewBotRegistration(messenger string) bool {
	mutexInterface, ok := connector.messengersCache.Get(messenger)
	if !ok {
		return false
	}
	mutex := mutexInterface.(*redsync.Mutex)
	return mutex.Extend()
}

// DeregisterBots cancels registration for all registered messengers
func (connector *DbConnector) DeregisterBots() {
	messengers := connector.messengersCache.Items()
	for messenger := range messengers {
		connector.DeregisterBot(messenger)
	}
}

// DeregisterBot removes registration of bot instance in redis
func (connector *DbConnector) DeregisterBot(messenger string) bool {
	mutexInterface, ok := connector.messengersCache.Get(messenger)
	if !ok {
		return false
	}
	mutex := mutexInterface.(*redsync.Mutex)
	return mutex.Unlock()
}

func usernameKey(messenger, username string) string {
	return fmt.Sprintf("moira-%s-users:%s", messenger, username)
}
