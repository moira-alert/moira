package redis

import (
	"testing"

	"github.com/moira-alert/moira"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"

	. "github.com/smartystreets/goconvey/convey"
)

func TestCursor(t *testing.T) {
	logger, _ := logging.GetLogger("database")
	db := newTestDatabase(logger, config)
	db.flush()
	cursor := db.NewCursor("MATCH", "moira-metric-data:*")

	Convey("Prepare data", t, func() {
		metric := "my.test.super.metric"
		metricValues := &moira.MatchedMetric{
			Patterns:           []string{"my.test.*.metric*"},
			Metric:             metric,
			Retention:          10,
			RetentionTimestamp: 10,
			Timestamp:          15,
			Value:              1,
		}
		err := db.SaveMetrics(map[string]*moira.MatchedMetric{metric: metricValues})
		So(err, ShouldBeNil)
	})

	Convey("Found metric key via cursor", t, func() {
		data, err := cursor.Next()
		So(err, ShouldBeNil)
		So(data, ShouldNotBeNil)
		So(len(data), ShouldEqual, 1)
	})

	Convey("Cursor return error on next when end of collection", t, func() {
		data, err := cursor.Next()
		So(err, ShouldNotBeNil)
		So(data, ShouldBeNil)
	})
}
