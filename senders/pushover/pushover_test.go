package pushover

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/gregdel/pushover"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/logging/go-logging"
	. "github.com/smartystreets/goconvey/convey"
)

func TestSender_Init(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "debug", "test")
	Convey("Empty map", t, func(c C) {
		sender := Sender{}
		err := sender.Init(map[string]string{}, logger, nil, "")
		c.So(err, ShouldResemble, fmt.Errorf("can not read pushover api_token from config"))
		c.So(sender, ShouldResemble, Sender{})
	})

	Convey("Settings has api_token", t, func(c C) {
		sender := Sender{}
		err := sender.Init(map[string]string{"api_token": "123"}, logger, nil, "")
		c.So(err, ShouldBeNil)
		c.So(sender, ShouldResemble, Sender{apiToken: "123", client: pushover.New("123"), logger: logger})
	})

	Convey("Settings has all data", t, func(c C) {
		sender := Sender{}
		location, _ := time.LoadLocation("UTC")
		err := sender.Init(map[string]string{"api_token": "123", "front_uri": "321"}, logger, location, "")
		c.So(err, ShouldBeNil)
		c.So(sender, ShouldResemble, Sender{apiToken: "123", client: pushover.New("123"), frontURI: "321", logger: logger, location: location})
	})
}

func TestGetPushoverPriority(t *testing.T) {
	sender := Sender{}
	Convey("All events has OK state", t, func(c C) {
		priority := sender.getMessagePriority([]moira.NotificationEvent{{State: moira.StateOK}, {State: moira.StateOK}, {State: moira.StateOK}})
		c.So(priority, ShouldResemble, pushover.PriorityNormal)
	})

	Convey("One of events has WARN state", t, func(c C) {
		priority := sender.getMessagePriority([]moira.NotificationEvent{{State: moira.StateOK}, {State: moira.StateWARN}, {State: moira.StateOK}})
		c.So(priority, ShouldResemble, pushover.PriorityHigh)
	})

	Convey("One of events has NODATA state", t, func(c C) {
		priority := sender.getMessagePriority([]moira.NotificationEvent{{State: moira.StateOK}, {State: moira.StateNODATA}, {State: moira.StateOK}})
		c.So(priority, ShouldResemble, pushover.PriorityHigh)
	})

	Convey("One of events has ERROR state", t, func(c C) {
		priority := sender.getMessagePriority([]moira.NotificationEvent{{State: moira.StateOK}, {State: moira.StateERROR}, {State: moira.StateOK}})
		c.So(priority, ShouldResemble, pushover.PriorityEmergency)
	})

	Convey("One of events has EXCEPTION state", t, func(c C) {
		priority := sender.getMessagePriority([]moira.NotificationEvent{{State: moira.StateOK}, {State: moira.StateEXCEPTION}, {State: moira.StateOK}})
		c.So(priority, ShouldResemble, pushover.PriorityEmergency)
	})

	Convey("Events has WARN and ERROR states", t, func(c C) {
		priority := sender.getMessagePriority([]moira.NotificationEvent{{State: moira.StateOK}, {State: moira.StateWARN}, {State: moira.StateERROR}})
		c.So(priority, ShouldResemble, pushover.PriorityEmergency)
	})

	Convey("Events has ERROR and WARN states", t, func(c C) {
		priority := sender.getMessagePriority([]moira.NotificationEvent{{State: moira.StateOK}, {State: moira.StateERROR}, {State: moira.StateWARN}})
		c.So(priority, ShouldResemble, pushover.PriorityEmergency)
	})
}

func TestBuildMoiraMessage(t *testing.T) {
	location, _ := time.LoadLocation("UTC")
	sender := Sender{location: location}
	value := float64(123)
	message := "This is message"

	Convey("Build Moira Message tests", t, func(c C) {
		event := moira.NotificationEvent{
			Value:     &value,
			Timestamp: 150000000,
			Metric:    "Metric",
			OldState:  moira.StateOK,
			State:     moira.StateNODATA,
			Message:   nil,
		}

		Convey("Print moira message with one event", t, func(c C) {
			actual := sender.buildMessage([]moira.NotificationEvent{event}, false)
			expected := "02:40: Metric = 123 (OK to NODATA)\n"
			c.So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with one event and message", t, func(c C) {
			event.Message = &message
			actual := sender.buildMessage([]moira.NotificationEvent{event}, false)
			expected := "02:40: Metric = 123 (OK to NODATA). This is message\n"
			c.So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with one event and throttled", t, func(c C) {
			actual := sender.buildMessage([]moira.NotificationEvent{event}, true)
			expected := `02:40: Metric = 123 (OK to NODATA)

Please, fix your system or tune this trigger to generate less events.`
			c.So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with 6 events", t, func(c C) {
			actual := sender.buildMessage([]moira.NotificationEvent{event, event, event, event, event, event}, false)
			expected := `02:40: Metric = 123 (OK to NODATA)
02:40: Metric = 123 (OK to NODATA)
02:40: Metric = 123 (OK to NODATA)
02:40: Metric = 123 (OK to NODATA)
02:40: Metric = 123 (OK to NODATA)

...and 1 more events.`
			c.So(actual, ShouldResemble, expected)
		})
	})
}

func TestBuildTitle(t *testing.T) {
	sender := Sender{}
	Convey("Build title with three events with max ERROR state and two tags", t, func(c C) {
		title := sender.buildTitle([]moira.NotificationEvent{{State: moira.StateERROR}, {State: moira.StateWARN}, {State: moira.StateWARN}, {State: moira.StateOK}}, moira.TriggerData{Tags: []string{"tag1", "tag2"}, Name: "Name"})
		c.So(title, ShouldResemble, "ERROR Name [tag1][tag2] (4)")
	})
	Convey("Build title with three events with max ERROR state empty trigger", t, func(c C) {
		title := sender.buildTitle([]moira.NotificationEvent{{State: moira.StateERROR}, {State: moira.StateWARN}, {State: moira.StateWARN}, {State: moira.StateOK}}, moira.TriggerData{})
		c.So(title, ShouldResemble, "ERROR   (4)")
	})
	Convey("Build title that exceeds the title limit", t, func(c C) {
		var reallyLongTag string
		for i := 0; i < 30; i++ {
			reallyLongTag = reallyLongTag + "randomstring"
		}
		title := sender.buildTitle([]moira.NotificationEvent{{State: moira.StateERROR}, {State: moira.StateWARN}, {State: moira.StateWARN}, {State: moira.StateOK}}, moira.TriggerData{Tags: []string{"tag1", "tag2", "tag3", reallyLongTag, "tag4"}, Name: "Name"})
		c.So(title, ShouldResemble, "ERROR Name [tag1][tag2][tag3].... (4)")
	})
}

func TestMakePushoverMessage(t *testing.T) {
	location, _ := time.LoadLocation("UTC")
	logger, _ := logging.ConfigureLog("stdout", "debug", "test")

	value := float64(123)
	sender := Sender{
		frontURI: "https://my-moira.com",
		location: location,
		logger:   logger,
	}
	Convey("Just build PushoverMessage", t, func(c C) {
		event := []moira.NotificationEvent{{
			Value:     &value,
			Timestamp: 150000000,
			Metric:    "Metric",
			OldState:  moira.StateOK,
			State:     moira.StateERROR,
			Message:   nil,
		},
		}
		trigger := moira.TriggerData{
			ID:   "SomeID",
			Name: "TriggerName",
			Tags: []string{"tag1", "tag2"},
		}
		contact := moira.ContactData{
			Value: "123",
		}
		expected := &pushover.Message{
			Timestamp: 150000000,
			Retry:     5 * time.Minute,
			Expire:    time.Hour,
			URL:       "https://my-moira.com/trigger/SomeID",
			Priority:  pushover.PriorityEmergency,
			Title:     "ERROR TriggerName [tag1][tag2] (1)",
			Message:   "02:40: Metric = 123 (OK to ERROR)\n",
		}
		expected.AddAttachment(bytes.NewReader([]byte{1, 0, 1}))
		c.So(sender.makePushoverMessage(event, contact, trigger, []byte{1, 0, 1}, false), ShouldResemble, expected)
	})
}
