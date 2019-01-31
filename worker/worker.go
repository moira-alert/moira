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
				worker.logger.Errorf("%s failed to acquire lock: %s", worker.name, err.Error())

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
		actionWg := &sync.WaitGroup{}
		actionWg.Add(1)
		go func(action func(<-chan struct{}), actionWg *sync.WaitGroup, actionStop <-chan struct{}) {
			action(actionStop)
			actionWg.Done()
		}(worker.action, actionWg, actionStop)

		select {
		case <-lost:
			worker.logger.Warningf("%s lost the lock", worker.name)
			close(actionStop)
			actionWg.Wait()
			worker.lock.Release()
		case <-stop:
			close(actionStop)
			actionWg.Wait()
			worker.lock.Release()
			return
		}
	}
}
