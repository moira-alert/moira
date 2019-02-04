package worker

import (
	"testing"
	"time"

	"github.com/pkg/errors"

	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database"
	"github.com/moira-alert/moira/mock/moira-alert"
	"github.com/op/go-logging"
	. "github.com/smartystreets/goconvey/convey"
)

const testLockRetryDelay = time.Millisecond * 100

func Test(t *testing.T) {
	Convey("Should stop if the lock's acquire was interrupted", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		lock := mock_moira_alert.NewMockLock(mockCtrl)
		worker := createTestWorkerWithDefaultAction(lock)
		stop := make(chan struct{})

		lock.EXPECT().Acquire(gomock.Any()).Return(nil, database.ErrLockAcquireInterrupted).Do(func(_ interface{}) { close(stop) })
		worker.Run(stop)
	})

	Convey("Should try to reacquire the lock with delay", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		stop := make(chan struct{})
		lock := mock_moira_alert.NewMockLock(mockCtrl)
		worker := createTestWorkerWithDefaultAction(lock)

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
		worker := createTestWorkerWithDefaultAction(lock)

		lock.EXPECT().Acquire(gomock.Any()).Return(nil, database.ErrLockNotAcquired).Do(func(_ interface{}) { close(stop) })

		worker.Run(stop)
	})

	Convey("Should handle error and release the lock", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		stop := make(chan struct{})
		lock := mock_moira_alert.NewMockLock(mockCtrl)
		worker := createTestWorkerWithAction(lock, func(stop <-chan struct{}) error { return errors.New("Oops") })
		gomock.InOrder(
			lock.EXPECT().Acquire(gomock.Any()).Return(nil, nil),
			lock.EXPECT().Release(),
			lock.EXPECT().Acquire(gomock.Any()).Return(nil, database.ErrLockAcquireInterrupted).Do(func(_ interface{}) { close(stop) }),
		)
		worker.Run(stop)
	})

	Convey("Should recover panic and release the lock", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		stop := make(chan struct{})
		lock := mock_moira_alert.NewMockLock(mockCtrl)
		worker := createTestWorkerWithAction(lock, func(stop <-chan struct{}) error { panic("Oops") })
		gomock.InOrder(
			lock.EXPECT().Acquire(gomock.Any()).Return(nil, nil),
			lock.EXPECT().Release(),
			lock.EXPECT().Acquire(gomock.Any()).Return(nil, database.ErrLockAcquireInterrupted).Do(func(_ interface{}) { close(stop) }),
		)
		worker.Run(stop)
	})

	Convey("Worker should respect lost chanel", t, func() {

		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		lock := mock_moira_alert.NewMockLock(mockCtrl)
		worker := createTestWorkerWithDefaultAction(lock)
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

		worker := createTestWorkerWithDefaultAction(lock)
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

func createTestWorkerWithDefaultAction(lock moira.Lock) *Worker {
	worker := NewWorker(
		"Test Worker",
		logging.MustGetLogger("Test Worker"),
		lock,
		func(stop <-chan struct{}) error {
			<-stop
			return nil
		},
	)
	worker.SetLockRetryDelay(testLockRetryDelay)
	return worker
}

func createTestWorkerWithAction(lock moira.Lock, action Action) *Worker {
	worker := NewWorker(
		"Test Worker",
		logging.MustGetLogger("Test Worker"),
		lock,
		action,
	)
	worker.SetLockRetryDelay(testLockRetryDelay)
	return worker
}
