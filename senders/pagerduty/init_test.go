package pagerduty

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"

	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	. "github.com/smartystreets/goconvey/convey"
)

const pagerdutyType = "pagerduty"

func TestInit(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "debug", "test", true)
	location, _ := time.LoadLocation("UTC")
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	imageStore := mock_moira_alert.NewMockImageStore(mockCtrl)

	Convey("Init tests", t, func() {
		sender := Sender{}

		Convey("Has settings", func() {
			imageStore.EXPECT().IsEnabled().Return(true)
			senderSettings := map[string]interface{}{
				"type":        pagerdutyType,
				"front_uri":   "http://moira.uri",
				"image_store": "s3",
			}

			opts := moira.InitOptions{
				SenderSettings: senderSettings,
				Logger:         logger,
				Location:       location,
				Database:       dataBase,
				ImageStores: map[string]moira.ImageStore{
					"s3": imageStore,
				},
			}

			err := sender.Init(opts)
			So(err, ShouldBeNil)

			client := sender.clients[pagerdutyType]
			So(client, ShouldNotBeNil)
			So(client.frontURI, ShouldResemble, "http://moira.uri")
			So(client.logger, ShouldResemble, logger)
			So(client.location, ShouldResemble, location)
			So(client.imageStoreConfigured, ShouldResemble, true)
			So(client.imageStore, ShouldResemble, imageStore)
		})

		Convey("Wrong image_store name", func() {
			senderSettings := map[string]interface{}{
				"type":        pagerdutyType,
				"front_uri":   "http://moira.uri",
				"image_store": "s4",
			}

			opts := moira.InitOptions{
				SenderSettings: senderSettings,
				Logger:         logger,
				Location:       location,
				Database:       dataBase,
				ImageStores: map[string]moira.ImageStore{
					"s3": imageStore,
				},
			}

			err := sender.Init(opts)
			So(err, ShouldBeNil)

			client := sender.clients[pagerdutyType]
			So(client, ShouldNotBeNil)
			So(client.frontURI, ShouldResemble, "http://moira.uri")
			So(client.imageStoreConfigured, ShouldResemble, false)
			So(client.imageStore, ShouldResemble, nil)
		})

		Convey("image store not configured", func() {
			imageStore.EXPECT().IsEnabled().Return(false)
			senderSettings := map[string]interface{}{
				"type":        pagerdutyType,
				"front_uri":   "http://moira.uri",
				"image_store": "s3",
			}

			opts := moira.InitOptions{
				SenderSettings: senderSettings,
				Logger:         logger,
				Location:       location,
				Database:       dataBase,
				ImageStores: map[string]moira.ImageStore{
					"s3": imageStore,
				},
			}

			err := sender.Init(opts)
			So(err, ShouldBeNil)

			client := sender.clients[pagerdutyType]
			So(client, ShouldNotBeNil)
			So(client.imageStoreConfigured, ShouldResemble, false)
			So(client.imageStore, ShouldResemble, nil)
		})
	})
}
