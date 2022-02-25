package database

import "fmt"

// ErrNil return from database data storing methods if no object in DB
var ErrNil = fmt.Errorf("nil returned")

var (
	// ErrLockAlreadyHeld is returned if we attempt to double acquire
	ErrLockAlreadyHeld = fmt.Errorf("lock was already held")
	// ErrLockAcquireInterrupted is returned if we cancel the acquire
	ErrLockAcquireInterrupted = fmt.Errorf("lock's request was interrupted")
)

// ErrLockNotAcquired if we cannot acquire
type ErrLockNotAcquired struct {
	Err error
}

func (e *ErrLockNotAcquired) Error() string {
	return fmt.Sprintf("lock was not acquired: %v", e.Err)
}
