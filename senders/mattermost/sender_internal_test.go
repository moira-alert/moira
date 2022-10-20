package mattermost

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/moira-alert/moira"

	"github.com/mattermost/mattermost-server/v6/model"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	. "github.com/smartystreets/goconvey/convey"
)

func TestSendEvents(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "debug", "test", true)
	sender := &Sender{}

	Convey("Given configured sender", t, func() {
		senderSettings := map[string]string{"url": "qwerty", "api_token": "qwerty", "front_uri": "qwerty"} // redundant, but necessary config
		err := sender.Init(senderSettings, logger, nil, "")
		So(err, ShouldBeNil)

		Convey("When client return error, SendEvents should return error", func() {
			sender.client = &TestClient4Error{}

			events, contact, trigger, plots, throttled := moira.NotificationEvents{}, moira.ContactData{}, moira.TriggerData{}, make([][]byte, 0), false
			err = sender.SendEvents(events, contact, trigger, plots, throttled)
			So(err, ShouldNotBeNil)
		})

		Convey("When client CreatePost is success, SendEvents should not return error", func() {
			sender.client = &TestClient4{}

			events, contact, trigger, plots, throttled := moira.NotificationEvents{}, moira.ContactData{}, moira.TriggerData{}, make([][]byte, 0), false
			err = sender.SendEvents(events, contact, trigger, plots, throttled)
			So(err, ShouldBeNil)
		})
	})
}

type TestClient4Error struct {
	Client
}

func (c *TestClient4Error) CreatePost(*model.Post) (*model.Post, *model.Response, error) {
	return nil, nil, errors.New("")
}

type TestClient4 struct {
	Client
}

func (c *TestClient4) CreatePost(*model.Post) (*model.Post, *model.Response, error) {
	return nil, nil, nil
}

func TestBuildMessage(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "debug", "test", true)
	sender := &Sender{}

	Convey("Given configured sender", t, func() {
		senderSettings := map[string]string{
			"url": "qwerty", "api_token": "qwerty", // redundant, but necessary config
			"front_uri": "http://moira.url",
		}
		location, _ := time.LoadLocation("UTC")
		err := sender.Init(senderSettings, logger, location, "")
		So(err, ShouldBeNil)

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
			msg := sender.buildMessage(events, trigger, throttled)

			expected := "**NODATA** [Name](http://moira.url/trigger/TriggerID) [tag1][tag2]\n" +
				shortDesc + "\n" +
				"```\n" +
				"02:40: Metric = 123 (OK to NODATA)```"
			So(msg, ShouldEqual, expected)
		})

		Convey("Message with one event and throttled", func() {
			events, throttled := moira.NotificationEvents{event}, true
			msg := sender.buildMessage(events, trigger, throttled)

			expected := "**NODATA** [Name](http://moira.url/trigger/TriggerID) [tag1][tag2]\n" +
				shortDesc + "\n" +
				"```\n" +
				"02:40: Metric = 123 (OK to NODATA)```" + "\n" +
				"Please, *fix your system or tune this trigger* to generate less events."
			So(msg, ShouldEqual, expected)
		})

		Convey("Moira message with 3 events", func() {
			actual := sender.buildMessage([]moira.NotificationEvent{event, event, event}, trigger, false)
			expected := "**NODATA** [Name](http://moira.url/trigger/TriggerID) [tag1][tag2]\n" +
				shortDesc + "\n" +
				"```\n" +
				"02:40: Metric = 123 (OK to NODATA)\n" +
				"02:40: Metric = 123 (OK to NODATA)\n" +
				"02:40: Metric = 123 (OK to NODATA)```"
			So(actual, ShouldResemble, expected)
		})

		Convey("Long message parts", func() {
			const messageMaxCharacters = 4_000

			const eventLine = "\n02:40: Metric = 123 (OK to NODATA)"
			oneEventLineLen := len([]rune(eventLine))

			longDesc := strings.Repeat("a", messageMaxCharacters/2+100)

			// Events list with chars greater than half the message limit
			var longEvents moira.NotificationEvents
			for i := 0; i < (messageMaxCharacters/2+200)/oneEventLineLen; i++ {
				longEvents = append(longEvents, event)
			}

			Convey("Long description. desc > msgLimit/2", func() {
				var events moira.NotificationEvents
				for i := 0; i < (messageMaxCharacters/2-10)/oneEventLineLen; i++ {
					events = append(events, event)
				}

				actual := sender.buildMessage(events, moira.TriggerData{Desc: longDesc}, false)
				expected := "**NODATA**\n" +
					strings.Repeat("a", 2013) + "...\n" +
					"```\n" +
					strings.Repeat("02:40: Metric = 123 (OK to NODATA)\n", 55) +
					"02:40: Metric = 123 (OK to NODATA)```"
				So(actual, ShouldResemble, expected)
			})

			Convey("Many events. eventString > msgLimit/2", func() {
				desc := strings.Repeat("a", messageMaxCharacters/2-100)
				actual := sender.buildMessage(longEvents, moira.TriggerData{Desc: desc}, false)
				expected := "**NODATA**\n" +
					desc + "\n" +
					"```\n" +
					strings.Repeat("02:40: Metric = 123 (OK to NODATA)\n", 57) +
					"02:40: Metric = 123 (OK to NODATA)```\n" +
					"...and 4 more events."
				So(actual, ShouldResemble, expected)
			})

			Convey("Long description and many events. both desc and events > msgLimit/2", func() {
				actual := sender.buildMessage(longEvents, moira.TriggerData{Desc: longDesc}, false)
				expected := "**NODATA**\n" +
					strings.Repeat("a", 1984) + "...\n" +
					"```\n" +
					strings.Repeat("02:40: Metric = 123 (OK to NODATA)\n", 55) +
					"02:40: Metric = 123 (OK to NODATA)```\n" +
					"...and 6 more events."
				So(actual, ShouldResemble, expected)
			})
		})
	})
}
