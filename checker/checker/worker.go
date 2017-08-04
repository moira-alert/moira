package checker

import (
	"github.com/moira-alert/moira-alert"
	"sync"
)

type Worker struct {
	checkerNumber int
	logger        moira.Logger
	database      moira.Database
}

func NewChecker(checkerNumber int, logger moira.Logger, database moira.Database) *Worker {
	return &Worker{
		checkerNumber: checkerNumber,
		logger:        logger,
		database:      database,
	}
}

func (worker *Worker) Run(shutdown chan bool, wg *sync.WaitGroup) {
	panic("implement me")
}
