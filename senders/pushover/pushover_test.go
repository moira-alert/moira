package pushover

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	pushover_client "github.com/gregdel/pushover"
	"github.com/moira-alert/moira"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	. "github.com/smartystreets/goconvey/convey"
)

func TestInit(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "debug", "test", true)

	validatorErr := validator.ValidationErrors{}

	Convey("Empty map", t, func() {
		sender := Sender{}
		senderSettings := map[string]interface{}{}

		err := sender.Init(senderSettings, logger, nil, "")
		So(errors.As(err, &validatorErr), ShouldBeTrue)
		So(sender, ShouldResemble, Sender{})
	})

	Convey("Settings has api_token", t, func() {
		sender := Sender{}
		senderSettings := map[string]interface{}{
			"api_token": "123",
		}

		err := sender.Init(senderSettings, logger, nil, "")
		So(err, ShouldBeNil)
		So(sender, ShouldResemble, Sender{apiToken: "123", client: pushover_client.New("123"), logger: logger})
	})

	Convey("Settings has all data", t, func() {
		sender := Sender{}
		senderSettings := map[string]interface{}{
			"api_token": "123",
			"front_uri": "321",
		}
		location, _ := time.LoadLocation("UTC")

		err := sender.Init(senderSettings, logger, location, "")
		So(err, ShouldBeNil)
		So(sender, ShouldResemble, Sender{apiToken: "123", client: pushover_client.New("123"), frontURI: "321", logger: logger, location: location})
	})
}

func TestGetPushoverPriority(t *testing.T) {
	sender := Sender{}

	Convey("All events has OK state", t, func() {
		priority := sender.getMessagePriority([]moira.NotificationEvent{{State: moira.StateOK}, {State: moira.StateOK}, {State: moira.StateOK}})
		So(priority, ShouldResemble, pushover_client.PriorityNormal)
	})

	Convey("One of events has WARN state", t, func() {
		priority := sender.getMessagePriority([]moira.NotificationEvent{{State: moira.StateOK}, {State: moira.StateWARN}, {State: moira.StateOK}})
		So(priority, ShouldResemble, pushover_client.PriorityHigh)
	})

	Convey("One of events has NODATA state", t, func() {
		priority := sender.getMessagePriority([]moira.NotificationEvent{{State: moira.StateOK}, {State: moira.StateNODATA}, {State: moira.StateOK}})
		So(priority, ShouldResemble, pushover_client.PriorityHigh)
	})

	Convey("One of events has ERROR state", t, func() {
		priority := sender.getMessagePriority([]moira.NotificationEvent{{State: moira.StateOK}, {State: moira.StateERROR}, {State: moira.StateOK}})
		So(priority, ShouldResemble, pushover_client.PriorityEmergency)
	})

	Convey("One of events has EXCEPTION state", t, func() {
		priority := sender.getMessagePriority([]moira.NotificationEvent{{State: moira.StateOK}, {State: moira.StateEXCEPTION}, {State: moira.StateOK}})
		So(priority, ShouldResemble, pushover_client.PriorityEmergency)
	})

	Convey("Events has WARN and ERROR states", t, func() {
		priority := sender.getMessagePriority([]moira.NotificationEvent{{State: moira.StateOK}, {State: moira.StateWARN}, {State: moira.StateERROR}})
		So(priority, ShouldResemble, pushover_client.PriorityEmergency)
	})

	Convey("Events has ERROR and WARN states", t, func() {
		priority := sender.getMessagePriority([]moira.NotificationEvent{{State: moira.StateOK}, {State: moira.StateERROR}, {State: moira.StateWARN}})
		So(priority, ShouldResemble, pushover_client.PriorityEmergency)
	})
}

func TestBuildMoiraMessage(t *testing.T) {
	location, _ := time.LoadLocation("UTC")
	sender := Sender{location: location}

	Convey("Build Moira Message tests", t, func() {
		event := moira.NotificationEvent{
			Values:    map[string]float64{"t1": 123},
			Timestamp: 150000000,
			Metric:    "Metric",
			OldState:  moira.StateOK,
			State:     moira.StateNODATA,
		}

		Convey("Print moira message with one event", func() {
			actual := sender.buildMessage([]moira.NotificationEvent{event}, false)
			expected := "02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n"
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with one event and message", func() {
			var interval int64 = 24
			event.MessageEventInfo = &moira.EventInfo{Interval: &interval}
			actual := sender.buildMessage([]moira.NotificationEvent{event}, false)
			expected := "02:40 (GMT+00:00): Metric = 123 (OK to NODATA). This metric has been in bad state for more than 24 hours - please, fix.\n"
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with one event and throttled", func() {
			actual := sender.buildMessage([]moira.NotificationEvent{event}, true)
			expected := `02:40 (GMT+00:00): Metric = 123 (OK to NODATA)

Please, fix your system or tune this trigger to generate less events.`
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with 6 events", func() {
			actual := sender.buildMessage([]moira.NotificationEvent{event, event, event, event, event, event}, false)
			expected := `02:40 (GMT+00:00): Metric = 123 (OK to NODATA)
02:40 (GMT+00:00): Metric = 123 (OK to NODATA)
02:40 (GMT+00:00): Metric = 123 (OK to NODATA)
02:40 (GMT+00:00): Metric = 123 (OK to NODATA)
02:40 (GMT+00:00): Metric = 123 (OK to NODATA)

...and 1 more events.`
			So(actual, ShouldResemble, expected)
		})
	})
}

func TestBuildTitle(t *testing.T) {
	sender := Sender{}

	Convey("Build title with three events with max ERROR state and two tags without throttling", t, func() {
		title := sender.buildTitle([]moira.NotificationEvent{{State: moira.StateERROR}, {State: moira.StateWARN}, {State: moira.StateWARN}, {State: moira.StateOK}}, moira.TriggerData{Tags: []string{"tag1", "tag2"}, Name: "Name"}, false)
		So(title, ShouldResemble, "ERROR Name [tag1][tag2] (4)")
	})

	Convey("Build title with three events with last OK state and two tags when throttling", t, func() {
		title := sender.buildTitle([]moira.NotificationEvent{{State: moira.StateERROR}, {State: moira.StateWARN}, {State: moira.StateWARN}, {State: moira.StateOK}}, moira.TriggerData{Tags: []string{"tag1", "tag2"}, Name: "Name"}, true)
		So(title, ShouldResemble, "OK Name [tag1][tag2] (4)")
	})

	Convey("Build title with three events with max ERROR state empty trigger without throttling", t, func() {
		title := sender.buildTitle([]moira.NotificationEvent{{State: moira.StateERROR}, {State: moira.StateWARN}, {State: moira.StateWARN}, {State: moira.StateOK}}, moira.TriggerData{}, false)
		So(title, ShouldResemble, "ERROR   (4)")
	})

	Convey("Build title with three events with last OK state empty trigger when throttling", t, func() {
		title := sender.buildTitle([]moira.NotificationEvent{{State: moira.StateERROR}, {State: moira.StateWARN}, {State: moira.StateWARN}, {State: moira.StateOK}}, moira.TriggerData{}, true)
		So(title, ShouldResemble, "OK   (4)")
	})

	Convey("Build title that exceeds the title limit", t, func() {
		var reallyLongTag string
		for i := 0; i < 30; i++ {
			reallyLongTag += "randomstring"
		}

		Convey("without throttling", func() {
			title := sender.buildTitle([]moira.NotificationEvent{{State: moira.StateERROR}, {State: moira.StateWARN}, {State: moira.StateWARN}, {State: moira.StateOK}}, moira.TriggerData{Tags: []string{"tag1", "tag2", "tag3", reallyLongTag, "tag4"}, Name: "Name"}, false)
			So(title, ShouldResemble, "ERROR Name [tag1][tag2][tag3].... (4)")
		})

		Convey("when throttling", func() {
			title := sender.buildTitle([]moira.NotificationEvent{{State: moira.StateERROR}, {State: moira.StateWARN}, {State: moira.StateWARN}, {State: moira.StateOK}}, moira.TriggerData{Tags: []string{"tag1", "tag2", "tag3", reallyLongTag, "tag4"}, Name: "Name"}, true)
			So(title, ShouldResemble, "OK Name [tag1][tag2][tag3].... (4)")
		})
	})
}

func TestMakePushoverMessage(t *testing.T) {
	location, _ := time.LoadLocation("UTC")
	logger, _ := logging.ConfigureLog("stdout", "debug", "test", true)

	sender := Sender{
		frontURI: "https://my-moira.com",
		location: location,
		logger:   logger,
	}

	Convey("Just build PushoverMessage", t, func() {
		event := []moira.NotificationEvent{
			{
				Values:    map[string]float64{"t1": 123},
				Timestamp: 150000000,
				Metric:    "Metric",
				OldState:  moira.StateOK,
				State:     moira.StateERROR,
			},
		}
		trigger := moira.TriggerData{
			ID:   "SomeID",
			Name: "TriggerName",
			Tags: []string{"tag1", "tag2"},
		}
		expected := &pushover_client.Message{
			Timestamp: 150000000,
			Retry:     5 * time.Minute,
			Expire:    time.Hour,
			URL:       "https://my-moira.com/trigger/SomeID",
			Priority:  pushover_client.PriorityEmergency,
			Title:     "ERROR TriggerName [tag1][tag2] (1)",
			Message:   "02:40 (GMT+00:00): Metric = 123 (OK to ERROR)\n",
		}
		expected.AddAttachment(bytes.NewReader([]byte{1, 0, 1})) //nolint
		So(sender.makePushoverMessage(event, trigger, [][]byte{{1, 0, 1}}, false), ShouldResemble, expected)
	})
}
