package redis

import (
	"github.com/op/go-logging"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
	"time"
)

func Test(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	database := newTestDatabase(logger, config)
	database.flush()
	defer database.flush()

	Convey("Simple lock scenario", t, func() {
		lock := database.NewLock("test", time.Second)
		_, err := lock.Acquire(nil)
		So(err, ShouldBeNil)
		lock.Release()
	})
}
