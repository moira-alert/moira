package database

import (
	"fmt"
)

// ErrDatabase used if database error occurred
type ErrDatabase struct {
	Err error
}

// ErrDatabase implementation with error message
func (err ErrDatabase) Error() string {
	return fmt.Sprintf("error: %s", err.Err.Error())
}
