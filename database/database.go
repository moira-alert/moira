package database

import "fmt"

// ErrNil return from database data storing methods if no object in DB
var ErrNil = fmt.Errorf("nil returned")

var (
	ErrLockAlreadyHeld        = fmt.Errorf("lock was already held")
	ErrLockAcquireInterrupted = fmt.Errorf("lock's request was interrupted")
	ErrLockNotAcquired        = fmt.Errorf("lock was not acquired")
)
