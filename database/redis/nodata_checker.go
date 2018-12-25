package redis

import (
	"time"

	"github.com/patrickmn/go-cache"
	"gopkg.in/redsync.v1"
)

// RegisterNodataCheckerIfAlreadyNot creates registration of NODATA checker instance in redis
func (connector *DbConnector) RegisterNodataCheckerIfAlreadyNot(ttl time.Duration) bool {
	mutex := connector.sync.NewMutex(nodataCheckerNameKey, redsync.SetExpiry(ttl), redsync.SetTries(1))
	if err := mutex.Lock(); err != nil {
		return false
	}
	connector.servicesCache.Set(nodataCheckerNameKey, mutex, cache.NoExpiration)
	return true
}

// RenewNodataCheckerRegistration extends NODATA checker lock registrations for given ttl
func (connector *DbConnector) RenewNodataCheckerRegistration() bool {
	mutexInterface, ok := connector.servicesCache.Get(nodataCheckerNameKey)
	if !ok {
		return false
	}
	mutex := mutexInterface.(*redsync.Mutex)
	return mutex.Extend()
}

// DeregisterNodataChecker removes registration of NODATA checker instance in redis
func (connector *DbConnector) DeregisterNodataChecker() bool {
	mutexInterface, ok := connector.servicesCache.Get(nodataCheckerNameKey)
	if !ok {
		return false
	}
	mutex := mutexInterface.(*redsync.Mutex)
	return mutex.Unlock()
}

const nodataCheckerNameKey = "moira-nodata-checker"
