package mattermost

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/mattermost/mattermost/server/public/model"

	"github.com/moira-alert/moira"

	"github.com/golang/mock/gomock"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	mock "github.com/moira-alert/moira/mock/notifier/mattermost"
	. "github.com/smartystreets/goconvey/convey"
)

const mattermostType = "mattermost"

func TestSendEvents(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "debug", "test", true)
	sender := &Sender{
		clients: map[string]*mattermostClient{
			mattermostType: {},
		},
	}

	Convey("Given configured sender", t, func() {
		senderSettings := map[string]interface{}{ // redundant, but necessary config
			"type":         mattermostType,
			"url":          "qwerty",
			"api_token":    "qwerty",
			"front_uri":    "qwerty",
			"insecure_tls": true,
		}

		opts := moira.InitOptions{
			SenderSettings: senderSettings,
			Logger:         logger,
			Location:       nil,
		}

		err := sender.Init(opts)
		So(err, ShouldBeNil)

		Convey("When client return error, SendEvents should return error", func() {
			ctrl := gomock.NewController(t)
			client := mock.NewMockClient(ctrl)
			client.EXPECT().CreatePost(context.Background(), gomock.Any()).Return(nil, nil, errors.New(""))
			mattermostClient := sender.clients[mattermostType]
			mattermostClient.client = client
			sender.clients[mattermostType] = mattermostClient

			events, contact, trigger, plots, throttled := moira.NotificationEvents{}, moira.ContactData{Type: mattermostType}, moira.TriggerData{}, make([][]byte, 0), false
			err = sender.SendEvents(events, contact, trigger, plots, throttled)
			So(err, ShouldNotBeNil)
		})

		Convey("When client CreatePost is success and no plots, SendEvents should not return error", func() {
			ctrl := gomock.NewController(t)
			client := mock.NewMockClient(ctrl)
			client.EXPECT().CreatePost(context.Background(), gomock.Any()).Return(&model.Post{Id: "postID"}, nil, nil)
			mattermostClient := sender.clients[mattermostType]
			mattermostClient.client = client
			sender.clients[mattermostType] = mattermostClient

			events, contact, trigger, plots, throttled := moira.NotificationEvents{}, moira.ContactData{Type: mattermostType}, moira.TriggerData{}, make([][]byte, 0), false
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
					}, nil, nil,
				)
			mattermostClient := sender.clients[mattermostType]
			mattermostClient.client = client
			sender.clients[mattermostType] = mattermostClient

			plots := make([][]byte, 0)
			plots = append(plots, []byte("my_awesome_plot"))
			events, contact, trigger, throttled := moira.NotificationEvents{}, moira.ContactData{Type: mattermostType, Value: "contactDataID"}, moira.TriggerData{ID: "triggerID"}, false
			err = sender.SendEvents(events, contact, trigger, plots, throttled)
			So(err, ShouldBeNil)
		})
	})
}

func TestBuildMessage(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "debug", "test", true)
	sender := &Sender{}

	Convey("Given configured sender", t, func() {
		senderSettings := map[string]interface{}{
			"type":         mattermostType,
			"url":          "qwerty",
			"api_token":    "qwerty", // redundant, but necessary config
			"front_uri":    "http://moira.url",
			"insecure_tls": true,
		}
		location, _ := time.LoadLocation("UTC")

		opts := moira.InitOptions{
			SenderSettings: senderSettings,
			Logger:         logger,
			Location:       location,
		}

		err := sender.Init(opts)
		So(err, ShouldBeNil)

		mattermostClient := sender.clients[mattermostType]

		event := moira.NotificationEvent{
			TriggerID: "TriggerID",
			Values:    map[string]float64{"t1": 123},
			Timestamp: 150000000,
			Metric:    "Metric",
			OldState:  moira.StateOK,
			State:     moira.StateNODATA,
		}

		const shortDesc = `My description`
		trigger := moira.TriggerData{
			Tags: []string{"tag1", "tag2"},
			Name: "Name",
			ID:   "TriggerID",
			Desc: shortDesc,
		}

		Convey("Message with one event", func() {
			events, throttled := moira.NotificationEvents{event}, false
			msg := mattermostClient.buildMessage(events, trigger, throttled)

			expected := "**NODATA** [Name](http://moira.url/trigger/TriggerID) [tag1][tag2]\n" +
				shortDesc + "\n" +
				"```\n" +
				"02:40 (GMT+00:00): Metric = 123 (OK to NODATA)```"
			So(msg, ShouldEqual, expected)
		})

		Convey("Message with one event and throttled", func() {
			events, throttled := moira.NotificationEvents{event}, true
			msg := mattermostClient.buildMessage(events, trigger, throttled)

			expected := "**NODATA** [Name](http://moira.url/trigger/TriggerID) [tag1][tag2]\n" +
				shortDesc + "\n" +
				"```\n" +
				"02:40 (GMT+00:00): Metric = 123 (OK to NODATA)```" + "\n" +
				"Please, *fix your system or tune this trigger* to generate less events."
			So(msg, ShouldEqual, expected)
		})

		Convey("Moira message with 3 events", func() {
			actual := mattermostClient.buildMessage([]moira.NotificationEvent{event, event, event}, trigger, false)
			expected := "**NODATA** [Name](http://moira.url/trigger/TriggerID) [tag1][tag2]\n" +
				shortDesc + "\n" +
				"```\n" +
				"02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n" +
				"02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n" +
				"02:40 (GMT+00:00): Metric = 123 (OK to NODATA)```"
			So(actual, ShouldResemble, expected)
		})

		Convey("Long message parts", func() {
			const (
				msgLimit        = 4_000
				halfLimit       = msgLimit / 2
				greaterThanHalf = halfLimit + 100
				lessThanHalf    = halfLimit - 100
			)

			const eventLine = "\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)"
			oneEventLineLen := len([]rune(eventLine))

			longDesc := strings.Repeat("a", greaterThanHalf)

			// Events list with chars greater than half of the message limit
			var longEvents moira.NotificationEvents
			for i := 0; i < greaterThanHalf/oneEventLineLen; i++ {
				longEvents = append(longEvents, event)
			}

			Convey("Long description. desc > msgLimit/2", func() {
				var events moira.NotificationEvents
				for i := 0; i < lessThanHalf/oneEventLineLen; i++ {
					events = append(events, event)
				}

				actual := mattermostClient.buildMessage(events, moira.TriggerData{Desc: longDesc}, false)
				expected := "**NODATA**\n" +
					strings.Repeat("a", 2100) + "\n" +
					"```\n" +
					strings.Repeat("02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n", 39) +
					"02:40 (GMT+00:00): Metric = 123 (OK to NODATA)```"
				So(actual, ShouldResemble, expected)
			})

			Convey("Many events. eventString > msgLimit/2", func() {
				desc := strings.Repeat("a", lessThanHalf)
				actual := mattermostClient.buildMessage(longEvents, moira.TriggerData{Desc: desc}, false)
				expected := "**NODATA**\n" +
					desc + "\n" +
					"```\n" +
					strings.Repeat("02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n", 43) +
					"02:40 (GMT+00:00): Metric = 123 (OK to NODATA)```"
				So(actual, ShouldResemble, expected)
			})

			Convey("Long description and many events. both desc and events > msgLimit/2", func() {
				actual := mattermostClient.buildMessage(longEvents, moira.TriggerData{Desc: longDesc}, false)
				expected := "**NODATA**\n" +
					strings.Repeat("a", 1984) + "...\n" +
					"```\n" +
					strings.Repeat("02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n", 40) +
					"02:40 (GMT+00:00): Metric = 123 (OK to NODATA)```\n" +
					"...and 3 more events."
				So(actual, ShouldResemble, expected)
			})
		})
	})
}
