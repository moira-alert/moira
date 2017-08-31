package database

import "fmt"

// ErrNil return from database data storing methods if no object in DB
var ErrNil = fmt.Errorf("Nil returned")
