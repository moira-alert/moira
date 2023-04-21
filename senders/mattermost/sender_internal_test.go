package mattermost

import (
	"errors"
	"testing"

	"github.com/moira-alert/moira"

	"github.com/golang/mock/gomock"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	mock "github.com/moira-alert/moira/mock/notifier/mattermost"
	. "github.com/smartystreets/goconvey/convey"
)

func TestSendEvents(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "debug", "test", true)
	sender := &Sender{}

	Convey("Given configured sender", t, func() {
		senderSettings := map[string]string{ // redundant, but necessary config
			"url":          "qwerty",
			"api_token":    "qwerty",
			"front_uri":    "qwerty",
			"insecure_tls": "true",
		}
		err := sender.Init(senderSettings, logger, nil, "")
		So(err, ShouldBeNil)

		Convey("When client return error, SendEvents should return error", func() {
			ctrl := gomock.NewController(t)
			client := mock.NewMockClient(ctrl)
			client.EXPECT().CreatePost(gomock.Any()).Return(nil, nil, errors.New(""))
			sender.client = client

			events, contact, trigger, plots, throttled := moira.NotificationEvents{}, moira.ContactData{}, moira.TriggerData{}, make([][]byte, 0), false
			err = sender.SendEvents(events, contact, trigger, plots, throttled)
			So(err, ShouldNotBeNil)
		})

		Convey("When client CreatePost is success, SendEvents should not return error", func() {
			ctrl := gomock.NewController(t)
			client := mock.NewMockClient(ctrl)
			client.EXPECT().CreatePost(gomock.Any()).Return(nil, nil, nil)
			sender.client = client

			events, contact, trigger, plots, throttled := moira.NotificationEvents{}, moira.ContactData{}, moira.TriggerData{}, make([][]byte, 0), false
			err = sender.SendEvents(events, contact, trigger, plots, throttled)
			So(err, ShouldBeNil)
		})
	})
}
