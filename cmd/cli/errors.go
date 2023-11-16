package main

import (
	"fmt"
	"reflect"

	"github.com/moira-alert/moira"
)

type unknownDBError struct {
	database reflect.Type
}

func makeUnknownDBError(database moira.Database) unknownDBError {
	return unknownDBError{
		database: reflect.TypeOf(database),
	}
}

func (err unknownDBError) Error() string {
	return fmt.Sprintf("Unknown implementation of moira.Database: %s", err.database.Name())
}
