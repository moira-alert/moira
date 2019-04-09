package redis

import (
	"testing"

	"github.com/op/go-logging"
	. "github.com/smartystreets/goconvey/convey"

	"time"
)

func TestThrottlingErrorConnection(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := newTestDatabase(logger, emptyConfig)
	dataBase.flush()
	defer dataBase.flush()
	Convey("Should throw error when no connection", t, func(c C) {
		t1, t2 := dataBase.GetTriggerThrottling("")
		c.So(t1, ShouldResemble, time.Unix(0, 0))
		c.So(t2, ShouldResemble, time.Unix(0, 0))

		err := dataBase.SetTriggerThrottling("", time.Now())
		c.So(err, ShouldNotBeNil)

		err = dataBase.DeleteTriggerThrottling("")
		c.So(err, ShouldNotBeNil)
	})
}
