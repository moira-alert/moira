package main

import (
	"fmt"
	"reflect"

	"github.com/moira-alert/moira"
)

type UnknownDBError struct {
	database reflect.Type
}

func MakeUnknownDBError(database moira.Database) UnknownDBError {
	return UnknownDBError{
		database: reflect.TypeOf(database),
	}
}

func (err UnknownDBError) Error() string {
	return fmt.Sprintf("Unknown implementation of moira.Database: %s", err.database.Name())
}
