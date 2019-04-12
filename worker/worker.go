package worker

import (
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database"
)

const lockRetryDelay = time.Second * 5

// Action is the shorthand to `func(stop <-chan struct{}) error`
type Action func(stop <-chan struct{}) error

// NewWorker creates Worker
func NewWorker(name string, logger moira.Logger, lock moira.Lock, action Action) *Worker {
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
	action         Action
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
				worker.logger.Errorf("%s failed to acquire the lock: %s", worker.name, err.Error())
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
		actionDone := make(chan struct{})
		go func(action Action, logger moira.Logger, done chan struct{}, stop <-chan struct{}) {
			defer close(done)

			defer func() {
				if r := recover(); r != nil {
					logger.Errorf("%s panicked during the execution: %s", worker.name, r)
				}
			}()

			if err := action(stop); err != nil {
				logger.Errorf("%s failed during the execution: %s", worker.name, err.Error())
			}
		}(worker.action, worker.logger, actionDone, actionStop)

		select {
		case <-actionDone:
			worker.lock.Release()
			worker.logger.Infof("%s released the lock", worker.name)
			select {
			case <-stop:
				return
			case <-time.After(worker.lockRetryDelay):
				continue
			}
		case <-lost:
			worker.logger.Warningf("%s lost the lock", worker.name)
			close(actionStop)
			<-actionDone
			worker.lock.Release()
		case <-stop:
			close(actionStop)
			<-actionDone
			worker.lock.Release()
			worker.logger.Infof("%s released the lock", worker.name)
			return
		}
	}
}
