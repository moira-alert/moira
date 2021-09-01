package goredis

import (
	"context"

	"github.com/go-redis/redis/v8"
	"github.com/moira-alert/moira"
)

// DBSource is type for describing who create database instance
type DBSource string

// All types of database users
const (
	API        DBSource = "API"
	Checker    DBSource = "Checker"
	Filter     DBSource = "Filter"
	Notifier   DBSource = "Notifier"
	Cli        DBSource = "Cli"
	testSource DBSource = "test"
)

// DbConnector contains redis client
type DbConnector struct {
	client  *redis.UniversalClient
	logger  moira.Logger
	context context.Context
	source  DBSource
}

func NewDatabase(logger moira.Logger, config Config, source DBSource) *DbConnector {
	client := redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs: config.Addrs,
	})

	ctx := context.Background()

	connector := DbConnector{
		client:  &client,
		logger:  logger,
		context: ctx,
		source:  source,
	}
	return &connector
}

// Deletes all the keys of the DB, use it only for tests
func (connector *DbConnector) flush() {
	(*connector.client).FlushDB(connector.context)
}
