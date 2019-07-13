package pagerduty

import (
	"fmt"
	"testing"
	"time"

	"github.com/moira-alert/moira/logging/go-logging"
	. "github.com/smartystreets/goconvey/convey"
)

func TestInit(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "debug", "test")
	location, _ := time.LoadLocation("UTC")
	Convey("Init tests", t, func() {
		sender := Sender{}

		Convey("Missing image_store", func() {
			senderSettings := map[string]string{
				"front_uri": "http://moira.uri",
			}
			err := sender.Init(senderSettings, logger, location, "15:04")
			So(err, ShouldResemble, fmt.Errorf("cannot read image_store from the config"))
		})
		Convey("Has settings", func() {
			senderSettings := map[string]string{
				"front_uri":   "http://moira.uri",
				"image_store": "s3",
			}
			sender.Init(senderSettings, logger, location, "15:04")
			So(sender.frontURI, ShouldResemble, "http://moira.uri")
			So(sender.logger, ShouldResemble, logger)
			So(sender.location, ShouldResemble, location)
		})
	})
}
