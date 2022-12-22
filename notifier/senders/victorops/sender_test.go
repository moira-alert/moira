package victorops

import (
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/notifier/senders/victorops/api"

	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	. "github.com/smartystreets/goconvey/convey"
)

func TestNewSender(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "debug", "test", true)
	location, _ := time.LoadLocation("UTC")
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	Convey("Init tests", t, func() {
		Convey("Empty map", func() {
			imageStore := mock_moira_alert.NewMockImageStore(mockCtrl)
			imageStores := map[string]moira.ImageStore{
				"s3": imageStore,
			}
			sender, err := NewSender(map[string]string{}, logger, nil, imageStores)
			So(err, ShouldResemble, fmt.Errorf("cannot read the routing url from the yaml config"))
			So(sender, ShouldBeNil)
		})

		Convey("Has settings", func() {
			imageStore := mock_moira_alert.NewMockImageStore(mockCtrl)
			imageStore.EXPECT().IsEnabled().Return(true)
			imageStores := map[string]moira.ImageStore{
				"s3": imageStore,
			}
			senderSettings := map[string]string{
				"routing_url": "https://testurl.com",
				"front_uri":   "http://moira.uri",
				"image_store": "s3",
			}
			sender, err := NewSender(senderSettings, logger, location, imageStores)
			So(err, ShouldBeNil)
			So(sender.routingURL, ShouldResemble, "https://testurl.com")
			So(sender.frontURI, ShouldResemble, "http://moira.uri")
			So(sender.logger, ShouldResemble, logger)
			So(sender.location, ShouldResemble, location)
			So(sender.client, ShouldResemble, api.NewClient("https://testurl.com", nil))
		})

		Convey("Wrong image_store name", func() {
			imageStore := mock_moira_alert.NewMockImageStore(mockCtrl)
			imageStores := map[string]moira.ImageStore{
				"s3": imageStore,
			}
			senderSettings := map[string]string{
				"front_uri":   "http://moira.uri",
				"routing_url": "https://testurl.com",
				"image_store": "s4",
			}
			sender, err := NewSender(senderSettings, logger, location, imageStores)
			So(err, ShouldBeNil)
			So(sender.imageStoreConfigured, ShouldResemble, false)
			So(sender.imageStore, ShouldResemble, nil)
		})

		Convey("image store not configured", func() {
			imageStore := mock_moira_alert.NewMockImageStore(mockCtrl)
			imageStore.EXPECT().IsEnabled().Return(false)
			imageStores := map[string]moira.ImageStore{
				"s3": imageStore,
			}
			senderSettings := map[string]string{
				"front_uri":   "http://moira.uri",
				"routing_url": "https://testurl.com",
				"image_store": "s3",
			}
			sender, err := NewSender(senderSettings, logger, location, imageStores)
			So(err, ShouldBeNil)
			So(sender.imageStoreConfigured, ShouldResemble, false)
			So(sender.imageStore, ShouldResemble, nil)
		})
	})
}
