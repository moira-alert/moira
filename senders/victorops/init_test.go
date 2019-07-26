package victorops

import (
	"fmt"
	"testing"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/senders/victorops/api"

	"github.com/moira-alert/moira/logging/go-logging"
	. "github.com/smartystreets/goconvey/convey"
)

type MockImageStoreNotConfigured struct{ moira.ImageStore }

func (imageStore *MockImageStoreNotConfigured) IsEnabled() bool { return false }

func TestInit(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "debug", "test")
	location, _ := time.LoadLocation("UTC")
	Convey("Init tests", t, func() {
		sender := Sender{ImageStores: map[string]moira.ImageStore{
			"s3": &MockImageStore{},
		}}
		Convey("Empty map", func() {
			err := sender.Init(map[string]string{}, logger, nil, "")
			So(err, ShouldResemble, fmt.Errorf("cannot read the routing url from the yaml config"))
			So(sender, ShouldResemble, Sender{
				ImageStores: map[string]moira.ImageStore{
					"s3": &MockImageStore{},
				}})
		})

		Convey("Has settings", func() {
			senderSettings := map[string]string{
				"routing_url": "https://testurl.com",
				"front_uri":   "http://moira.uri",
			}
			sender.Init(senderSettings, logger, location, "15:04")
			So(sender.routingURL, ShouldResemble, "https://testurl.com")
			So(sender.frontURI, ShouldResemble, "http://moira.uri")
			So(sender.logger, ShouldResemble, logger)
			So(sender.location, ShouldResemble, location)
			So(sender.client, ShouldResemble, api.NewClient("https://testurl.com", nil))
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
		Convey("image store not configured", func() {
			senderSettings := map[string]string{
				"front_uri":   "http://moira.uri",
				"image_store": "s3",
			}
			sender := Sender{ImageStores: map[string]moira.ImageStore{
				"s3": &MockImageStoreNotConfigured{},
			}}
			sender.Init(senderSettings, logger, location, "15:04")
			So(sender.imageStoreConfigured, ShouldResemble, false)
			So(sender.imageStore, ShouldResemble, nil)
		})

	})
}
