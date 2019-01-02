package redis

import (
	"fmt"
	"github.com/moira-alert/moira"
	"time"

	"github.com/patrickmn/go-cache"
	"gopkg.in/redsync.v1"
)

// RegisterServiceIfNotDone creates registration of service instance in redis.
// It is useful for services that should run in the only instance.
// it can be:
// - selfState checker
// - NODATA checker
// - telegram bot
func (connector *DbConnector) RegisterServiceIfNotDone(service moira.SingleInstanceService, ttl time.Duration) bool {
	mutex := connector.sync.NewMutex(serviceRegistrationKey(service), redsync.SetExpiry(ttl), redsync.SetTries(1))
	if err := mutex.Lock(); err != nil {
		return false
	}
	connector.servicesCache.Set(serviceRegistrationKey(service), mutex, cache.NoExpiration)
	return true
}

// RenewServiceRegistration extends service lock registrations for given ttl
func (connector *DbConnector) RenewServiceRegistration(service moira.SingleInstanceService) bool {
	mutexInterface, ok := connector.servicesCache.Get(serviceRegistrationKey(service))
	if !ok {
		return false
	}
	mutex := mutexInterface.(*redsync.Mutex)
	return mutex.Extend()
}

// DeregisterService removes registration of service instance in redis
func (connector *DbConnector) DeregisterService(service moira.SingleInstanceService) bool {
	mutexInterface, ok := connector.servicesCache.Get(serviceRegistrationKey(service))
	if !ok {
		return false
	}
	mutex := mutexInterface.(*redsync.Mutex)
	return mutex.Unlock()
}

func serviceRegistrationKey(service moira.SingleInstanceService) string {
	return fmt.Sprintf("moira-service-registration:%s", service)
}
