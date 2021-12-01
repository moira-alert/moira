package redis

import (
	"testing"

	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	. "github.com/smartystreets/goconvey/convey"

	"time"
)

func TestThrottlingErrorConnection(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := NewTestDatabaseWithIncorrectConfig(logger)
	dataBase.Flush()
	defer dataBase.Flush()
	Convey("Should throw error when no connection", t, func() {
		t1, t2 := dataBase.GetTriggerThrottling("")
		So(t1, ShouldResemble, time.Unix(0, 0))
		So(t2, ShouldResemble, time.Unix(0, 0))

		err := dataBase.SetTriggerThrottling("", time.Now())
		So(err, ShouldNotBeNil)

		err = dataBase.DeleteTriggerThrottling("")
		So(err, ShouldNotBeNil)
	})
}
