package worker

import (
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database"
	"sync"
	"time"
)

const lockRetryDelay = time.Second * 5

// NewWorker creates Worker
func NewWorker(name string, logger moira.Logger, lock moira.Lock, action func(stop <-chan struct{})) *Worker {
	return &Worker{name: name, logger: logger, lock: lock, action: action, lockRetryDelay: lockRetryDelay}
}

// SetLockRetryDelay changes the delay between unsuccessful acquire attempts of the lock
func (worker *Worker) SetLockRetryDelay(lockRetryDelay time.Duration) {
	worker.lockRetryDelay = lockRetryDelay
}

// Worker simplifies usage of the lock
type Worker struct {
	name           string
	logger         moira.Logger
	lock           moira.Lock
	action         func(stop <-chan struct{})
	wg             sync.WaitGroup
	lockRetryDelay time.Duration
}

// Run the worker
func (worker *Worker) Run(stop <-chan struct{}) {
	for {
		worker.logger.Infof("%s tries to acquire the lock...", worker.name)
		lost, err := worker.lock.Acquire(stop)
		if err != nil {
			switch err {
			case database.ErrLockAcquireInterrupted:
				return
			default:
				worker.logger.Errorf("%s failed to acquire lock %s", worker.name, err.Error())

				select {
				case <-stop:
					return
				case <-time.After(worker.lockRetryDelay):
					continue
				}
			}
		}

		worker.logger.Infof("%s acquired the lock", worker.name)

		actionStop := make(chan struct{})
		worker.wg.Add(1)
		go func(worker *Worker, actionStop <-chan struct{}) {
			worker.action(actionStop)
			worker.lock.Release()
			worker.wg.Done()
		}(worker, actionStop)

		select {
		case <-lost:
			worker.logger.Warningf("%s lost the lock", worker.name)
			close(actionStop)
			worker.wg.Wait()
		case <-stop:
			close(actionStop)
			worker.wg.Wait()
			return
		}
	}
}
