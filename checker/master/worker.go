package master

import (
	"github.com/moira-alert/moira-alert"
	"sync"
)

type Worker struct {
	logger   moira.Logger
	database moira.Database
}

func NewMaster(logger moira.Logger, database moira.Database) *Worker {
	return &Worker{
		logger:   logger,
		database: database,
	}
}

func (worker *Worker) Run(shutdown chan bool, wg *sync.WaitGroup) {
	panic("implement me")
}
