package checker

import (
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/metrics/graphite"
	"sync"
)

type Worker struct {
	checkerNumber int
	logger        moira.Logger
	database      moira.Database
	metrics       *graphite.CheckerMetrics
}

func NewChecker(checkerNumber int, logger moira.Logger, database moira.Database, metrics *graphite.CheckerMetrics) *Worker {
	return &Worker{
		checkerNumber: checkerNumber,
		logger:        logger,
		database:      database,
		metrics:       metrics,
	}
}

func (worker *Worker) Run(shutdown chan bool, wg *sync.WaitGroup) {
	panic("implement me")
}
