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
		sender := Sender{ImageStores: map[string]moira.ImageStore{
			"s3": imageStore,
		}}

		Convey("Empty map", func() {
			sendersNameToType := make(map[string]string)
			err := sender.Init(map[string]interface{}{}, logger, nil, "", sendersNameToType)
			So(err, ShouldResemble, fmt.Errorf("cannot read the api_key from the sender settings"))
			So(sender, ShouldResemble, Sender{
				ImageStores: map[string]moira.ImageStore{
					"s3": imageStore,
				},
			})
		})

		Convey("Has settings", func() {
			imageStore.EXPECT().IsEnabled().Return(true)
			senderSettings := map[string]interface{}{
				"type":        opsgenieType,
				"api_key":     "testkey",
				"front_uri":   "http://moira.uri",
				"image_store": "s3",
			}
			sendersNameToType := make(map[string]string)

			err := sender.Init(senderSettings, logger, location, "15:04", sendersNameToType)
			So(err, ShouldBeNil)
			So(sender.apiKey, ShouldResemble, "testkey")
			So(sender.frontURI, ShouldResemble, "http://moira.uri")
			So(sender.logger, ShouldResemble, logger)
			So(sender.location, ShouldResemble, location)
			So(sendersNameToType[opsgenieType], ShouldEqual, senderSettings["type"])
		})

		Convey("Wrong image_store name", func() {
			senderSettings := map[string]interface{}{
				"type":        opsgenieType,
				"front_uri":   "http://moira.uri",
				"api_key":     "testkey",
				"image_store": "s4",
			}
			sendersNameToType := make(map[string]string)

			err := sender.Init(senderSettings, logger, location, "15:04", sendersNameToType)
			So(err, ShouldBeNil)
			So(sender.imageStoreConfigured, ShouldResemble, false)
			So(sender.imageStore, ShouldResemble, nil)
		})

		Convey("image store not configured", func() {
			imageStore.EXPECT().IsEnabled().Return(false)
			senderSettings := map[string]interface{}{
				"type":        opsgenieType,
				"api_key":     "testkey",
				"front_uri":   "http://moira.uri",
				"image_store": "s3",
			}
			sender := Sender{ImageStores: map[string]moira.ImageStore{
				"s3": imageStore,
			}}
			sendersNameToType := make(map[string]string)

			err := sender.Init(senderSettings, logger, location, "15:04", sendersNameToType)
			So(err, ShouldBeNil)
			So(sender.imageStoreConfigured, ShouldResemble, false)
			So(sender.imageStore, ShouldResemble, nil)
			So(sendersNameToType[opsgenieType], ShouldEqual, senderSettings["type"])
		})
	})
}
