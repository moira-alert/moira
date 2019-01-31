package worker

import (
	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira/database"
	"github.com/moira-alert/moira/mock/moira-alert"
	"github.com/op/go-logging"
	. "github.com/smartystreets/goconvey/convey"
	"sync"
	"testing"
	"time"
)

func Test(t *testing.T) {
	logger, _ := logging.GetLogger("Worker")

	Convey("Should stop", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		lock := mock_moira_alert.NewMockLock(mockCtrl)
		worker := NewWorker("test", logger, lock, func(stop <-chan struct{}) {})
		stop := make(chan struct{})

		lock.EXPECT().Acquire(gomock.Any()).Return(nil, database.ErrLockAcquireInterrupted)
		worker.Run(stop)
	})

	Convey("Should try to reacquire with delay", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		lockRetryDelay := time.Millisecond * 100
		lock := mock_moira_alert.NewMockLock(mockCtrl)
		worker := NewWorker("test", logger, lock, func(stop <-chan struct{}) {})
		worker.SetLockRetryDelay(lockRetryDelay)
		stop := make(chan struct{})

		lock.EXPECT().Acquire(gomock.Any()).Return(nil, database.ErrLockNotAcquired)
		lock.EXPECT().Acquire(gomock.Any()).Return(nil, database.ErrLockAcquireInterrupted)

		go func() {
			time.Sleep(lockRetryDelay)
			close(stop)
		}()

		worker.Run(stop)
	})

	Convey("Worker should respect lost chanel", t, func() {

		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		lock := mock_moira_alert.NewMockLock(mockCtrl)
		lost := make(chan struct{})
		worker := NewWorker("test", logger, lock, func(stop <-chan struct{}) {
			select {
			case <-stop:
				{
					return
				}
			}
		})
		stop := make(chan struct{})

		lock.EXPECT().Acquire(gomock.Any()).Return(lost, nil)
		lock.EXPECT().Release()
		lock.EXPECT().Acquire(gomock.Any()).Return(nil, database.ErrLockAcquireInterrupted)

		go close(lost)
		worker.Run(stop)
	})

	Convey("Worker should respect stop chanel", t, func() {

		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		lock := mock_moira_alert.NewMockLock(mockCtrl)
		lost := make(chan struct{})
		worker := NewWorker("test", logger, lock, func(stop <-chan struct{}) {
			select {
			case <-stop:
				{
					return
				}
			}
		})
		stop := make(chan struct{})

		lock.EXPECT().Acquire(gomock.Any()).Return(lost, nil)
		lock.EXPECT().Release()

		go close(stop)
		worker.Run(stop)
	})

	Convey("Worker should wait until action is finished", t, func() {

		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		lockRetryDelay := time.Millisecond * 100
		lock := mock_moira_alert.NewMockLock(mockCtrl)

		worker := NewWorker("test", logger, lock, func(stop <-chan struct{}) {
			select {
			case <-stop:
				time.Sleep(lockRetryDelay)
				return
			}
		})
		stop := make(chan struct{})

		lock.EXPECT().Acquire(gomock.Any()).Return(nil, nil)
		lock.EXPECT().Release()

		wg := &sync.WaitGroup{}
		wg.Add(1)

		start := time.Now()
		go func() {
			worker.Run(stop)
			wg.Done()
		}()
		close(stop)
		wg.Wait()

		So(time.Since(start), ShouldBeGreaterThanOrEqualTo, lockRetryDelay)
	})
}
