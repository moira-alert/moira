package victorops

import (
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/senders/victorops/api"

	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	. "github.com/smartystreets/goconvey/convey"
)

const victoropsType = "victorops"

func TestInit(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "debug", "test", true)
	location, _ := time.LoadLocation("UTC")
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	imageStore := mock_moira_alert.NewMockImageStore(mockCtrl)

	Convey("Init tests", t, func() {
		sender := Sender{ImageStores: map[string]moira.ImageStore{
			"s3": imageStore,
		}}

		Convey("Empty map", func() {
			err := sender.Init(map[string]interface{}{}, logger, nil, "")
			So(err, ShouldResemble, fmt.Errorf("cannot read the routing url from the yaml config"))
			So(sender, ShouldResemble, Sender{
				ImageStores: map[string]moira.ImageStore{
					"s3": imageStore,
				},
			})
		})

		Convey("Has settings", func() {
			imageStore.EXPECT().IsEnabled().Return(true)
			senderSettings := map[string]interface{}{
				"type":        victoropsType,
				"routing_url": "https://testurl.com",
				"front_uri":   "http://moira.uri",
				"image_store": "s3",
			}

			err := sender.Init(senderSettings, logger, location, "15:04")
			So(err, ShouldBeNil)
			So(sender.routingURL, ShouldResemble, "https://testurl.com")
			So(sender.frontURI, ShouldResemble, "http://moira.uri")
			So(sender.logger, ShouldResemble, logger)
			So(sender.location, ShouldResemble, location)
			So(sender.client, ShouldResemble, api.NewClient("https://testurl.com", nil))
		})

		Convey("Wrong image_store name", func() {
			senderSettings := map[string]interface{}{
				"type":        victoropsType,
				"front_uri":   "http://moira.uri",
				"routing_url": "https://testurl.com",
				"image_store": "s4",
			}

			err := sender.Init(senderSettings, logger, location, "15:04")
			So(err, ShouldBeNil)
			So(sender.imageStoreConfigured, ShouldResemble, false)
			So(sender.imageStore, ShouldResemble, nil)
		})

		Convey("image store not configured", func() {
			imageStore.EXPECT().IsEnabled().Return(false)
			senderSettings := map[string]interface{}{
				"type":        victoropsType,
				"front_uri":   "http://moira.uri",
				"routing_url": "https://testurl.com",
				"image_store": "s3",
			}
			sender := Sender{ImageStores: map[string]moira.ImageStore{
				"s3": imageStore,
			}}

			err := sender.Init(senderSettings, logger, location, "15:04")
			So(err, ShouldBeNil)
			So(sender.imageStoreConfigured, ShouldResemble, false)
			So(sender.imageStore, ShouldResemble, nil)
		})
	})
}
