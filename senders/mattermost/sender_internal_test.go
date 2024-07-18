package mattermost

import (
	"context"
	"errors"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"

	"github.com/moira-alert/moira"

	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	mock "github.com/moira-alert/moira/mock/notifier/mattermost"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/mock/gomock"
)

func TestSendEvents(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "debug", "test", true)
	sender := &Sender{}

	Convey("Given configured sender", t, func() {
		senderSettings := map[string]interface{}{ // redundant, but necessary config
			"url":          "qwerty",
			"api_token":    "qwerty",
			"front_uri":    "qwerty",
			"insecure_tls": true,
		}
		err := sender.Init(senderSettings, logger, nil, "")
		So(err, ShouldBeNil)

		Convey("When client return error, SendEvents should return error", func() {
			ctrl := gomock.NewController(t)
			client := mock.NewMockClient(ctrl)
			client.EXPECT().CreatePost(context.Background(), gomock.Any()).Return(nil, nil, errors.New(""))
			sender.client = client

			events, contact, trigger, plots, throttled := moira.NotificationEvents{}, moira.ContactData{}, moira.TriggerData{}, make([][]byte, 0), false
			err = sender.SendEvents(events, contact, trigger, plots, throttled)
			So(err, ShouldNotBeNil)
		})

		Convey("When client CreatePost is success and no plots, SendEvents should not return error", func() {
			ctrl := gomock.NewController(t)
			client := mock.NewMockClient(ctrl)
			client.EXPECT().CreatePost(context.Background(), gomock.Any()).Return(&model.Post{Id: "postID"}, nil, nil)
			sender.client = client

			events, contact, trigger, plots, throttled := moira.NotificationEvents{}, moira.ContactData{}, moira.TriggerData{}, make([][]byte, 0), false
			err = sender.SendEvents(events, contact, trigger, plots, throttled)
			So(err, ShouldBeNil)
		})

		Convey("When client CreatePost is success and have succeeded sent plots, SendEvents should not return error", func() {
			ctrl := gomock.NewController(t)
			client := mock.NewMockClient(ctrl)
			client.EXPECT().CreatePost(context.Background(), gomock.Any()).Return(&model.Post{Id: "postID"}, nil, nil).Times(2)
			client.EXPECT().UploadFile(context.Background(), gomock.Any(), "contactDataID", "triggerID.png").
				Return(
					&model.FileUploadResponse{
						FileInfos: []*model.FileInfo{{Id: "fileID"}},
					}, nil, nil)
			sender.client = client

			plots := make([][]byte, 0)
			plots = append(plots, []byte("my_awesome_plot"))
			events, contact, trigger, throttled := moira.NotificationEvents{}, moira.ContactData{Value: "contactDataID"}, moira.TriggerData{ID: "triggerID"}, false
			err = sender.SendEvents(events, contact, trigger, plots, throttled)
			So(err, ShouldBeNil)
		})
	})
}
