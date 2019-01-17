package redis

import (
	_ "fmt"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database"
	"gopkg.in/redsync.v1"
	"sync"
	"time"
)

func (connector *DbConnector) NewLock(name string, ttl time.Duration) moira.Lock {
	mutex := connector.sync.NewMutex(name, redsync.SetExpiry(ttl), redsync.SetTries(1))
	return &Lock{name: name, ttl: ttl, mutex: mutex}
}

type Lock struct {
	name   string
	ttl    time.Duration
	mutex  *redsync.Mutex
	extend chan struct{}
	m      sync.Mutex
	isHeld bool
}

func (lock *Lock) tryAcquire() (<-chan struct{}, error) {
	lock.m.Lock()
	defer lock.m.Unlock()

	if lock.isHeld {
		return nil, database.ErrLockAlreadyHeld
	}

	if err := lock.mutex.Lock(); err != nil {
		return nil, database.ErrLockNotAcquired
	}

	lost := make(chan struct{})
	lock.extend = make(chan struct{})
	go extendMutex(lock.mutex, lock.ttl, lost, lock.extend)
	lock.isHeld = true
	return lost, nil
}

func (lock *Lock) Acquire(stop <-chan struct{}) (<-chan struct{}, error) {

	for {
		lost, err := lock.tryAcquire()

		if err != nil {
			select {
			case <-stop:
				{
					return nil, database.ErrLockAcquireInterrupted
				}
			case <-time.After(lock.ttl / 3):
				{
					continue
				}
			}
		}

		return lost, nil
	}
}

func extendMutex(mutex *redsync.Mutex, ttl time.Duration, done chan struct{}, stop <-chan struct{}) {
	defer close(done)
	extendTicker := time.NewTicker(ttl / 3)
	defer extendTicker.Stop()

	for {
		select {
		case <-stop:
			{
				return
			}
		case <-extendTicker.C:
			if !mutex.Extend() {
				return
			}
		}
	}
}

func (lock *Lock) Release() {
	lock.m.Lock()
	defer lock.m.Unlock()

	if !lock.isHeld {
		return
	}

	lock.isHeld = false
	close(lock.extend)
	lock.mutex.Unlock()
}
