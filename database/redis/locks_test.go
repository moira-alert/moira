package redis

import (
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/moira-alert/moira/database"
	"github.com/op/go-logging"
	. "github.com/smartystreets/goconvey/convey"
)

func Test(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	db := newTestDatabase(logger, config)
	db.flush()
	defer db.flush()

	Convey("Acquire/Release", t, func(c C) {
		lockName := "test:" + strconv.Itoa(rand.Int())
		lock := db.NewLock(lockName, time.Second)
		_, err := lock.Acquire(nil)
		defer lock.Release()

		c.So(err, ShouldBeNil)
	})

	Convey("Background extent", t, func(c C) {
		lockName := "test:" + strconv.Itoa(rand.Int())
		lock := db.NewLock(lockName, time.Second)
		_, err := lock.Acquire(nil)
		defer lock.Release()
		c.So(err, ShouldBeNil)

		time.Sleep(2 * time.Second)

		c.So(db.getTTL(lockName), ShouldBeBetween, 0, time.Second)
	})

	Convey("Lost must be signalled", t, func(c C) {
		lockName := "test:" + strconv.Itoa(rand.Int())
		lock := db.NewLock(lockName, time.Second)
		lost, err := lock.Acquire(nil)
		defer lock.Release()
		c.So(err, ShouldBeNil)

		db.delete(lockName)

		isLost := func() bool {
			select {
			case <-lost:
				return true
			case <-time.After(time.Second):
				return false
			}
		}
		c.So(isLost(), ShouldBeTrue)
	})

	Convey("Can't double acquire on same lock", t, func(c C) {
		lockName := "test:" + strconv.Itoa(rand.Int())
		lock := db.NewLock(lockName, time.Second)
		_, err := lock.Acquire(nil)
		defer lock.Release()
		c.So(err, ShouldBeNil)

		_, err = lock.Acquire(nil)
		c.So(err, ShouldEqual, database.ErrLockAlreadyHeld)
	})
}
