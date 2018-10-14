package redis

import (
	"testing"
	"time"

	"github.com/op/go-logging"
	"github.com/patrickmn/go-cache"
	. "github.com/smartystreets/goconvey/convey"
	"gopkg.in/tomb.v2"

	"github.com/moira-alert/moira"
	"fmt"
)

func TestMatchedMetricsStoring(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := NewDatabase(logger, config)
	dataBase.flush()

	matchedValues := map[string]int64{
		"moira-docker1": 1200,
		"moira-docker2": 2500,
	}

	Convey("Manage matched metrics values", t, func() {
		var err error
		for source, matchedValue := range matchedValues {
			err = dataBase.SaveMatchedMetricsValue(source, 1, matchedValue)
			So(err, ShouldBeNil)
		}

		actualValues, err := dataBase.GetMatchedMetricsValues(0, 1)
		So(err, ShouldBeNil)
		fmt.Println(actualValues)

		err = dataBase.RemoveMatchedMetricsValues(1)
		So(err, ShouldBeNil)

		actualValues, err = dataBase.GetMatchedMetricsValues(0, 1)
		So(err, ShouldBeNil)
		fmt.Println(actualValues)
	})
}
