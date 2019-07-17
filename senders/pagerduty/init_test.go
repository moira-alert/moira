package pagerduty

import (
	"testing"
	"time"

	"github.com/moira-alert/moira"

	"github.com/moira-alert/moira/logging/go-logging"
	. "github.com/smartystreets/goconvey/convey"
)

func TestInit(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "debug", "test")
	location, _ := time.LoadLocation("UTC")
	Convey("Init tests", t, func() {
		sender := Sender{ImageStores: map[string]moira.ImageStore{
			"s3": &MockImageStore{},
		}}

		Convey("Has settings", func() {
			senderSettings := map[string]string{
				"front_uri":   "http://moira.uri",
				"image_store": "s3",
			}
			sender.Init(senderSettings, logger, location, "15:04")
			So(sender.frontURI, ShouldResemble, "http://moira.uri")
			So(sender.logger, ShouldResemble, logger)
			So(sender.location, ShouldResemble, location)
			So(sender.imageStoreConfigured, ShouldResemble, true)
			So(sender.imageStore, ShouldResemble, &MockImageStore{})
		})
		Convey("Wrong image_store name", func() {
			senderSettings := map[string]string{
				"front_uri":   "http://moira.uri",
				"image_store": "s4",
			}
			sender.Init(senderSettings, logger, location, "15:04")
			So(sender.imageStoreConfigured, ShouldResemble, false)
			So(sender.imageStore, ShouldResemble, nil)
		})

	})
}
