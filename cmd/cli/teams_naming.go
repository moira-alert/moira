package main

import (
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database/redis"
)

func fillTeamNamesHash(logger moira.Logger, database moira.Database) error {
	switch db := database.(type) {
	case *redis.DbConnector:
		_ = db
	default:
		return makeUnknownDBError(database)
	}
	return nil
}
