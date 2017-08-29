package redis

import (
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/metrics/graphite"
	"github.com/op/go-logging"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
	"time"
)

func TestEvents(t *testing.T) {
	logger, _ := logging.GetLogger("123")
	database := NewDatabase(logger, Config{Port: "6379", Host: "localhost"}, &graphite.DatabaseMetrics{})
	eventData := moira.NotificationEvent{
		Timestamp: time.Now().Unix(),
		State:     "NODATA",
		OldState:  "NODATA",
		TriggerID: "81588c33-eab3-4ad4-aa03-82a9560adad9",
		Metric:    "my.metric",
	}
	Convey("123", t, func() {
		err := database.PushEvent(&eventData, true)
		So(err, ShouldBeNil)
		events, err := database.GetEvents(eventData.TriggerID, 0, 0)
		So(err, ShouldBeNil)
		So(events, ShouldHaveLength, 1)
		So(events[0], ShouldResemble, &eventData)
	})
}
