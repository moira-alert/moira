package redis

import (
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/moira-alert/moira/database"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	. "github.com/smartystreets/goconvey/convey"
)

func Test(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	db := newTestDatabase(logger, config)
	db.flush()
	defer db.flush()

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

		So(db.getTTL(lockName), ShouldBeBetween, 0, time.Second)
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
}
