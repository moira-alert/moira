package redis

import (
	"errors"
	"sync"
	"time"

	"github.com/go-redsync/redsync/v4"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database"
)

// NewLock returns the implementation of moira.Lock which can be used to Acquire or Release the lock
func (connector *DbConnector) NewLock(name string, ttl time.Duration) moira.Lock {
	mutex := connector.sync.NewMutex(name, redsync.WithExpiry(ttl), redsync.WithTries(1))
	return &Lock{name: name, ttl: ttl, mutex: mutex}
}

// Lock is used to hide low-level details of redsync.Mutex such as an extension of it
type Lock struct {
	name   string
	ttl    time.Duration
	mutex  moira.Mutex
	extend chan struct{}
	m      sync.Mutex
	isHeld bool
}

// Acquire attempts to acquire the lock and blocks while doing so
// Providing a non-nil stop channel can be used to abort the acquire attempt
// Returns lost channel that is closed if the lock is lost or an error
func (lock *Lock) Acquire(stop <-chan struct{}) (<-chan struct{}, error) {
	for {
		lost, err := lock.tryAcquire()
		if err == nil {
			return lost, nil
		}

		if errors.Is(err, database.ErrLockAlreadyHeld) {
			return nil, database.ErrLockAlreadyHeld
		}

		switch e := err.(type) { // nolint:errorlint
		case *database.ErrLockNotAcquired:
			if !errors.Is(e.Err, redsync.ErrFailed) {
				return nil, err
			}
		}

		select {
		case <-stop:
			{
				return nil, database.ErrLockAcquireInterrupted
			}
		case <-time.After(lock.ttl / 3): //nolint
			{
				continue
			}
		}
	}
}

// Release releases the lock
func (lock *Lock) Release() {
	lock.m.Lock()
	defer lock.m.Unlock()

	if !lock.isHeld {
		return
	}

	lock.isHeld = false
	close(lock.extend)
	lock.mutex.Unlock() //nolint
}

func (lock *Lock) tryAcquire() (<-chan struct{}, error) {
	lock.m.Lock()
	defer lock.m.Unlock()

	if lock.isHeld {
		return nil, database.ErrLockAlreadyHeld
	}

	if err := lock.mutex.Lock(); err != nil {
		return nil, &database.ErrLockNotAcquired{Err: err}
	}

	lost := make(chan struct{})
	lock.extend = make(chan struct{})
	go extendMutex(lock.mutex, lock.ttl, lost, lock.extend)
	lock.isHeld = true
	return lost, nil
}

func extendMutex(mutex moira.Mutex, ttl time.Duration, done chan struct{}, stop <-chan struct{}) {
	defer close(done)
	extendTicker := time.NewTicker(ttl / 3) //nolint
	defer extendTicker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-extendTicker.C:
			result, _ := mutex.Extend()
			if !result {
				return
			}
		}
	}
}
