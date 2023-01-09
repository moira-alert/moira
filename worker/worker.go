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
		worker.logger.Infob().
			String("worker_name", worker.name).
			Msg("Worker tries to acquire the lock...")
		lost, err := worker.lock.Acquire(stop)
		if err != nil {
			switch err {
			case database.ErrLockAcquireInterrupted:
				return
			default:
				worker.logger.Errorb().
					String("worker_name", worker.name).
					Error(err).
					Msg("Worker failed to acquire the lock")
				select {
				case <-stop:
					return
				case <-time.After(worker.lockRetryDelay):
					continue
				}
			}
		}

		worker.logger.Infob().
			String("worker_name", worker.name).
			Msg("Worker acquired the lock")

		actionStop := make(chan struct{})
		actionDone := make(chan struct{})
		go func(action Action, logger moira.Logger, done chan struct{}, stop <-chan struct{}) {
			defer close(done)

			defer func() {
				if r := recover(); r != nil {
					logger.Errorb().
						String("worker_name", worker.name).
						Interface("recover", r).
						Msg("Worker panicked during the execution")
				}
			}()

			if err := action(stop); err != nil {
				logger.Errorb().
					String("worker_name", worker.name).
					Error(err).
					Msg("Worker failed during the execution")
			}
		}(worker.action, worker.logger, actionDone, actionStop)

		select {
		case <-actionDone:
			worker.lock.Release()
			worker.logger.Infob().
				String("worker_name", worker.name).
				Msg("Worker released the lock")
			select {
			case <-stop:
				return
			case <-time.After(worker.lockRetryDelay):
				continue
			}
		case <-lost:
			worker.logger.Warningb().
				String("worker_name", worker.name).
				Msg("Worker lost the lock")
			close(actionStop)
			<-actionDone
			worker.lock.Release()
		case <-stop:
			close(actionStop)
			<-actionDone
			worker.lock.Release()
			worker.logger.Infob().
				String("worker_name", worker.name).
				Msg("Worker released the lock")
			return
		}
	}
}
