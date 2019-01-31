package worker

import (
	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database"
	"github.com/moira-alert/moira/mock/moira-alert"
	"github.com/op/go-logging"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
	"time"
)

const testLockRetryDelay = time.Millisecond * 100

func Test(t *testing.T) {

	Convey("Should stop if the lock's acquire was interrupted", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		lock := mock_moira_alert.NewMockLock(mockCtrl)
		worker := createTestWorker(lock)
		stop := make(chan struct{})

		lock.EXPECT().Acquire(gomock.Any()).Return(nil, database.ErrLockAcquireInterrupted).Do(func(_ interface{}) { close(stop) })
		worker.Run(stop)
	})

	Convey("Should try to reacquire the lock with delay", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		stop := make(chan struct{})
		lock := mock_moira_alert.NewMockLock(mockCtrl)
		worker := createTestWorker(lock)

		gomock.InOrder(
			lock.EXPECT().Acquire(gomock.Any()).Return(nil, database.ErrLockNotAcquired),
			lock.EXPECT().Acquire(gomock.Any()).Return(nil, nil).Do(func(_ interface{}) { close(stop) }),
			lock.EXPECT().Release(),
		)

		start := time.Now()
		worker.Run(stop)
		So(time.Since(start), ShouldBeGreaterThanOrEqualTo, testLockRetryDelay)
	})

	Convey("Should interrupt the lock reacquire", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		stop := make(chan struct{})
		lock := mock_moira_alert.NewMockLock(mockCtrl)
		worker := createTestWorker(lock)

		lock.EXPECT().Acquire(gomock.Any()).Return(nil, database.ErrLockNotAcquired).Do(func(_ interface{}) { close(stop) })

		worker.Run(stop)
	})

	Convey("Worker should respect lost chanel", t, func() {

		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		lock := mock_moira_alert.NewMockLock(mockCtrl)
		worker := createTestWorker(lock)
		lost, stop := make(chan struct{}), make(chan struct{})

		gomock.InOrder(
			lock.EXPECT().Acquire(gomock.Any()).DoAndReturn(func(_ interface{}) (<-chan struct{}, error) {
				close(lost)
				return lost, nil
			}),
			lock.EXPECT().Release().Return(),
			lock.EXPECT().Acquire(gomock.Any()).Return(nil, database.ErrLockAcquireInterrupted).Do(func(_ interface{}) { close(stop) }),
		)
		worker.Run(stop)
	})

	Convey("Worker should respect stop chanel", t, func() {

		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		lock := mock_moira_alert.NewMockLock(mockCtrl)

		worker := createTestWorker(lock)
		stop := make(chan struct{})

		gomock.InOrder(
			lock.EXPECT().Acquire(gomock.Any()).DoAndReturn(func(_ interface{}) (<-chan struct{}, error) {
				close(stop)
				return nil, nil
			}),
			lock.EXPECT().Release().Return(),
		)

		worker.Run(stop)
	})
}

func createTestWorker(lock moira.Lock) *Worker {
	worker := NewWorker(
		"Test Worker",
		logging.MustGetLogger("Test Worker"),
		lock,
		func(stop <-chan struct{}) {
			select {
			case <-stop:
				{
					return
				}
			}
		})
	worker.SetLockRetryDelay(testLockRetryDelay)
	return worker
}
