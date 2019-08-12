package opsgenie

import (
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira"

	"github.com/moira-alert/moira/logging/go-logging"
	"github.com/moira-alert/moira/mock/moira-alert"
	. "github.com/smartystreets/goconvey/convey"
)

func TestInit(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "debug", "test")
	location, _ := time.LoadLocation("UTC")
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	imageStore := mock_moira_alert.NewMockImageStore(mockCtrl)

	Convey("Init tests", t, func() {
		sender := Sender{ImageStores: map[string]moira.ImageStore{
			"s3": imageStore,
		}}

		Convey("Empty map", func() {
			err := sender.Init(map[string]string{}, logger, nil, "")
			So(err, ShouldResemble, fmt.Errorf("cannot read the api_key from the sender settings"))
			So(sender, ShouldResemble, Sender{
				ImageStores: map[string]moira.ImageStore{
					"s3": imageStore,
				}})
		})

		Convey("Has settings", func() {
			imageStore.EXPECT().IsEnabled().Return(true)
			senderSettings := map[string]string{
				"api_key":     "testkey",
				"front_uri":   "http://moira.uri",
				"image_store": "s3",
			}
			sender.Init(senderSettings, logger, location, "15:04")
			So(sender.apiKey, ShouldResemble, "testkey")
			So(sender.frontURI, ShouldResemble, "http://moira.uri")
			So(sender.logger, ShouldResemble, logger)
			So(sender.location, ShouldResemble, location)
		})

		Convey("Wrong image_store name", func() {
			senderSettings := map[string]string{
				"front_uri":   "http://moira.uri",
				"api_key":     "testkey",
				"image_store": "s4",
			}
			sender.Init(senderSettings, logger, location, "15:04")
			So(sender.imageStoreConfigured, ShouldResemble, false)
			So(sender.imageStore, ShouldResemble, nil)
		})

		Convey("image store not configured", func() {
			imageStore.EXPECT().IsEnabled().Return(false)
			senderSettings := map[string]string{
				"api_key":     "testkey",
				"front_uri":   "http://moira.uri",
				"image_store": "s3",
			}
			sender := Sender{ImageStores: map[string]moira.ImageStore{
				"s3": imageStore,
			}}
			sender.Init(senderSettings, logger, location, "15:04")
			So(sender.imageStoreConfigured, ShouldResemble, false)
			So(sender.imageStore, ShouldResemble, nil)
		})

	})
}
