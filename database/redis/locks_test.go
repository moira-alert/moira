package redis

import (
	"strconv"

	"github.com/go-redsync/redsync/v4"

	"errors"

	"github.com/moira-alert/moira/database"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"

	"math/rand"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	. "github.com/smartystreets/goconvey/convey"
)

func Test(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	db := NewTestDatabase(logger)
	db.Flush()
	defer db.Flush()

	Convey("Acquire/Release", t, func() {
		lockName := "test:" + strconv.Itoa(rand.Int())
		lock := db.NewLock(lockName, time.Second)
		_, err := lock.Acquire(nil)
		defer lock.Release()

		So(err, ShouldBeNil)
	})

	Convey("Background extent", t, func() {
		lockName := "test:" + strconv.Itoa(rand.Int())
		lock := db.NewLock(lockName, time.Second)
		_, err := lock.Acquire(nil)
		defer lock.Release()
		So(err, ShouldBeNil)

		time.Sleep(2 * time.Second)
		So(db.getTTL(lockName), ShouldBeBetweenOrEqual, 0, time.Second)
	})

	Convey("Lost must be signalled", t, func() {
		lockName := "test:" + strconv.Itoa(rand.Int())
		lock := db.NewLock(lockName, time.Second)
		lost, err := lock.Acquire(nil)
		defer lock.Release()
		So(err, ShouldBeNil)

		db.delete(lockName)

		isLost := func() bool {
			select {
			case <-lost:
				return true
			case <-time.After(time.Second):
				return false
			}
		}
		So(isLost(), ShouldBeTrue)
	})

	Convey("Can't double acquire on same lock", t, func() {
		lockName := "test:" + strconv.Itoa(rand.Int())
		lock := db.NewLock(lockName, time.Second)
		_, err := lock.Acquire(nil)
		defer lock.Release()
		So(err, ShouldBeNil)

		_, err = lock.Acquire(nil)
		So(err, ShouldEqual, database.ErrLockAlreadyHeld)
	})

	Convey("ErrLockNotAcquired error is handled correctly", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		Convey("Lock returns ErrFailed", func() {
			mutex := mock_moira_alert.NewMockMutex(ctrl)

			gomock.InOrder(
				mutex.EXPECT().Lock().Return(redsync.ErrFailed),
				mutex.EXPECT().Lock().Return(nil),
				mutex.EXPECT().Unlock(),
			)

			lockName := "test:" + strconv.Itoa(rand.Int())
			lock := &Lock{name: lockName, ttl: time.Second, mutex: mutex}

			_, err := lock.Acquire(nil)
			defer lock.Release()
			So(err, ShouldBeNil)
		})

		Convey("Lock returns another error", func() {
			mutex := mock_moira_alert.NewMockMutex(ctrl)

			mutex.EXPECT().Lock().Return(errors.New("another error")).AnyTimes()

			lockName := "test:" + strconv.Itoa(rand.Int())
			lock := &Lock{name: lockName, ttl: time.Second, mutex: mutex}

			_, err := lock.Acquire(nil)
			defer lock.Release()
			So(err.Error(), ShouldEqual, "lock was not acquired: another error")
		})
	})
}
