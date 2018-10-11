package redis

import (
	"fmt"
	"time"

	"github.com/patrickmn/go-cache"
	"gopkg.in/redsync.v1"
)

// RegisterServiceIfAlreadyNot creates registration of Moira Service instance in redis
func (connector *DbConnector) RegisterServiceIfAlreadyNot(service, hostname string, ttl time.Duration) bool {
	mutex := connector.sync.NewMutex(serviceNameKey(service, hostname), redsync.SetExpiry(ttl), redsync.SetTries(1))
	if err := mutex.Lock(); err != nil {
		return false
	}
	connector.servicesCache.Set(hostname, mutex, cache.NoExpiration)
	return true
}

// RenewServiceRegistration extends Moira service lock registrations for given ttl
func (connector *DbConnector) RenewServiceRegistration(hostname string) bool {
	mutexInterface, ok := connector.servicesCache.Get(hostname)
	if !ok {
		return false
	}
	mutex := mutexInterface.(*redsync.Mutex)
	return mutex.Extend()
}

// DeregisterServices cancels registration for all registered services
func (connector *DbConnector) DeregisterServices() {
	hostnames := connector.servicesCache.Items()
	for hostname := range hostnames {
		connector.DeregisterService(hostname)
	}
}

// DeregisterService removes registration of service instance in redis
func (connector *DbConnector) DeregisterService(hostname string) bool {
	mutexInterface, ok := connector.servicesCache.Get(hostname)
	if !ok {
		return false
	}
	mutex := mutexInterface.(*redsync.Mutex)
	return mutex.Unlock()
}

func serviceNameKey(service, hostname string) string {
	return fmt.Sprintf("moira-%s-service:%s", service, hostname)
}
