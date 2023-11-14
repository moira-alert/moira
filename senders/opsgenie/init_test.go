package opsgenie

import (
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira"

	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	. "github.com/smartystreets/goconvey/convey"
)

const opsgenieType = "opsgenie"

func TestInit(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "debug", "test", true)
	location, _ := time.LoadLocation("UTC")
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	imageStore := mock_moira_alert.NewMockImageStore(mockCtrl)

	Convey("Init tests", t, func() {
		sender := Sender{}

		Convey("Empty map", func() {
			opts := moira.InitOptions{
				SenderSettings: map[string]interface{}{},
				Logger:         logger,
				Location:       nil,
				ImageStores: map[string]moira.ImageStore{
					"s3": imageStore,
				},
			}

			err := sender.Init(opts)
			So(err, ShouldResemble, fmt.Errorf("cannot read the api_key from the sender settings"))
		})

		Convey("Has settings", func() {
			imageStore.EXPECT().IsEnabled().Return(true)
			senderSettings := map[string]interface{}{
				"type":        opsgenieType,
				"api_key":     "testkey",
				"front_uri":   "http://moira.uri",
				"image_store": "s3",
			}

			opts := moira.InitOptions{
				SenderSettings: senderSettings,
				Logger:         logger,
				Location:       location,
				ImageStores: map[string]moira.ImageStore{
					"s3": imageStore,
				},
			}

			err := sender.Init(opts)
			So(err, ShouldBeNil)

			client := sender.clients[opsgenieType]
			So(client, ShouldNotBeNil)
			So(client.apiKey, ShouldResemble, "testkey")
			So(client.frontURI, ShouldResemble, "http://moira.uri")
			So(client.logger, ShouldResemble, logger)
			So(client.location, ShouldResemble, location)
		})

		Convey("Wrong image_store name", func() {
			senderSettings := map[string]interface{}{
				"type":        opsgenieType,
				"front_uri":   "http://moira.uri",
				"api_key":     "testkey",
				"image_store": "s4",
			}

			opts := moira.InitOptions{
				SenderSettings: senderSettings,
				Logger:         logger,
				Location:       location,
				ImageStores: map[string]moira.ImageStore{
					"s3": imageStore,
				},
			}

			err := sender.Init(opts)
			So(err, ShouldBeNil)

			client := sender.clients[opsgenieType]
			So(client, ShouldNotBeNil)
			So(client.imageStoreConfigured, ShouldResemble, false)
			So(client.imageStore, ShouldResemble, nil)
		})

		Convey("image store not configured", func() {
			imageStore.EXPECT().IsEnabled().Return(false)
			senderSettings := map[string]interface{}{
				"type":        opsgenieType,
				"api_key":     "testkey",
				"front_uri":   "http://moira.uri",
				"image_store": "s3",
			}

			opts := moira.InitOptions{
				SenderSettings: senderSettings,
				Logger:         logger,
				Location:       location,
				ImageStores: map[string]moira.ImageStore{
					"s3": imageStore,
				},
			}

			err := sender.Init(opts)
			So(err, ShouldBeNil)

			client := sender.clients[opsgenieType]
			So(client, ShouldNotBeNil)
			So(client.imageStoreConfigured, ShouldResemble, false)
			So(client.imageStore, ShouldResemble, nil)
		})
	})
}
