package pushover

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	pushover_client "github.com/gregdel/pushover"
	"github.com/moira-alert/moira"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	. "github.com/smartystreets/goconvey/convey"
)

const pushoverType = "pushover"

func TestSender_Init(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "debug", "test", true)

	Convey("Empty map", t, func() {
		sender := Sender{}
		opts := moira.InitOptions{
			SenderSettings: map[string]interface{}{},
			Logger:         logger,
			Location:       nil,
		}

		err := sender.Init(opts)
		So(err, ShouldResemble, fmt.Errorf("can not read pushover api_token from config"))
		So(sender, ShouldResemble, Sender{})
	})

	senderSettings := map[string]interface{}{
		"type": pushoverType,
	}

	Convey("Settings has api_token", t, func() {
		senderSettings["api_token"] = "123"
		sender := Sender{}
		opts := moira.InitOptions{
			SenderSettings: senderSettings,
			Logger:         logger,
			Location:       nil,
		}

		err := sender.Init(opts)
		So(err, ShouldBeNil)

		client := sender.clients[pushoverType]
		So(client, ShouldNotBeNil)
		So(client, ShouldResemble, &pushoverClient{
			apiToken: "123",
			client:   pushover_client.New("123"),
			logger:   logger,
		})
	})

	Convey("Settings has all data", t, func() {
		senderSettings["front_uri"] = "321"
		sender := Sender{}
		location, _ := time.LoadLocation("UTC")
		opts := moira.InitOptions{
			SenderSettings: senderSettings,
			Logger:         logger,
			Location:       location,
		}

		err := sender.Init(opts)
		So(err, ShouldBeNil)

		client := sender.clients[pushoverType]
		So(client, ShouldNotBeNil)
		So(client, ShouldResemble, &pushoverClient{
			apiToken: "123",
			client:   pushover_client.New("123"),
			frontURI: "321",
			logger:   logger,
			location: location,
		})
	})
}

func TestGetPushoverPriority(t *testing.T) {
	client := pushoverClient{}
	Convey("All events has OK state", t, func() {
		priority := client.getMessagePriority([]moira.NotificationEvent{{State: moira.StateOK}, {State: moira.StateOK}, {State: moira.StateOK}})
		So(priority, ShouldResemble, pushover_client.PriorityNormal)
	})

	Convey("One of events has WARN state", t, func() {
		priority := client.getMessagePriority([]moira.NotificationEvent{{State: moira.StateOK}, {State: moira.StateWARN}, {State: moira.StateOK}})
		So(priority, ShouldResemble, pushover_client.PriorityHigh)
	})

	Convey("One of events has NODATA state", t, func() {
		priority := client.getMessagePriority([]moira.NotificationEvent{{State: moira.StateOK}, {State: moira.StateNODATA}, {State: moira.StateOK}})
		So(priority, ShouldResemble, pushover_client.PriorityHigh)
	})

	Convey("One of events has ERROR state", t, func() {
		priority := client.getMessagePriority([]moira.NotificationEvent{{State: moira.StateOK}, {State: moira.StateERROR}, {State: moira.StateOK}})
		So(priority, ShouldResemble, pushover_client.PriorityEmergency)
	})

	Convey("One of events has EXCEPTION state", t, func() {
		priority := client.getMessagePriority([]moira.NotificationEvent{{State: moira.StateOK}, {State: moira.StateEXCEPTION}, {State: moira.StateOK}})
		So(priority, ShouldResemble, pushover_client.PriorityEmergency)
	})

	Convey("Events has WARN and ERROR states", t, func() {
		priority := client.getMessagePriority([]moira.NotificationEvent{{State: moira.StateOK}, {State: moira.StateWARN}, {State: moira.StateERROR}})
		So(priority, ShouldResemble, pushover_client.PriorityEmergency)
	})

	Convey("Events has ERROR and WARN states", t, func() {
		priority := client.getMessagePriority([]moira.NotificationEvent{{State: moira.StateOK}, {State: moira.StateERROR}, {State: moira.StateWARN}})
		So(priority, ShouldResemble, pushover_client.PriorityEmergency)
	})
}

func TestBuildMoiraMessage(t *testing.T) {
	location, _ := time.LoadLocation("UTC")
	client := pushoverClient{
		location: location,
	}

	Convey("Build Moira Message tests", t, func() {
		event := moira.NotificationEvent{
			Values:    map[string]float64{"t1": 123},
			Timestamp: 150000000,
			Metric:    "Metric",
			OldState:  moira.StateOK,
			State:     moira.StateNODATA,
		}

		Convey("Print moira message with one event", func() {
			actual := client.buildMessage([]moira.NotificationEvent{event}, false)
			expected := "02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n"
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with one event and message", func() {
			var interval int64 = 24
			event.MessageEventInfo = &moira.EventInfo{Interval: &interval}
			actual := client.buildMessage([]moira.NotificationEvent{event}, false)
			expected := "02:40 (GMT+00:00): Metric = 123 (OK to NODATA). This metric has been in bad state for more than 24 hours - please, fix.\n"
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with one event and throttled", func() {
			actual := client.buildMessage([]moira.NotificationEvent{event}, true)
			expected := `02:40 (GMT+00:00): Metric = 123 (OK to NODATA)

Please, fix your system or tune this trigger to generate less events.`
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with 6 events", func() {
			actual := client.buildMessage([]moira.NotificationEvent{event, event, event, event, event, event}, false)
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
	client := pushoverClient{}
	Convey("Build title with three events with max ERROR state and two tags without throttling", t, func() {
		title := client.buildTitle([]moira.NotificationEvent{{State: moira.StateERROR}, {State: moira.StateWARN}, {State: moira.StateWARN}, {State: moira.StateOK}}, moira.TriggerData{Tags: []string{"tag1", "tag2"}, Name: "Name"}, false)
		So(title, ShouldResemble, "ERROR Name [tag1][tag2] (4)")
	})

	Convey("Build title with three events with last OK state and two tags when throttling", t, func() {
		title := client.buildTitle([]moira.NotificationEvent{{State: moira.StateERROR}, {State: moira.StateWARN}, {State: moira.StateWARN}, {State: moira.StateOK}}, moira.TriggerData{Tags: []string{"tag1", "tag2"}, Name: "Name"}, true)
		So(title, ShouldResemble, "OK Name [tag1][tag2] (4)")
	})

	Convey("Build title with three events with max ERROR state empty trigger without throttling", t, func() {
		title := client.buildTitle([]moira.NotificationEvent{{State: moira.StateERROR}, {State: moira.StateWARN}, {State: moira.StateWARN}, {State: moira.StateOK}}, moira.TriggerData{}, false)
		So(title, ShouldResemble, "ERROR   (4)")
	})

	Convey("Build title with three events with last OK state empty trigger when throttling", t, func() {
		title := client.buildTitle([]moira.NotificationEvent{{State: moira.StateERROR}, {State: moira.StateWARN}, {State: moira.StateWARN}, {State: moira.StateOK}}, moira.TriggerData{}, true)
		So(title, ShouldResemble, "OK   (4)")
	})

	Convey("Build title that exceeds the title limit", t, func() {
		var reallyLongTag string
		for i := 0; i < 30; i++ {
			reallyLongTag = reallyLongTag + "randomstring"
		}

		Convey("without throttling", func() {
			title := client.buildTitle([]moira.NotificationEvent{{State: moira.StateERROR}, {State: moira.StateWARN}, {State: moira.StateWARN}, {State: moira.StateOK}}, moira.TriggerData{Tags: []string{"tag1", "tag2", "tag3", reallyLongTag, "tag4"}, Name: "Name"}, false)
			So(title, ShouldResemble, "ERROR Name [tag1][tag2][tag3].... (4)")
		})

		Convey("when throttling", func() {
			title := client.buildTitle([]moira.NotificationEvent{{State: moira.StateERROR}, {State: moira.StateWARN}, {State: moira.StateWARN}, {State: moira.StateOK}}, moira.TriggerData{Tags: []string{"tag1", "tag2", "tag3", reallyLongTag, "tag4"}, Name: "Name"}, true)
			So(title, ShouldResemble, "OK Name [tag1][tag2][tag3].... (4)")
		})
	})
}

func TestMakePushoverMessage(t *testing.T) {
	location, _ := time.LoadLocation("UTC")
	logger, _ := logging.ConfigureLog("stdout", "debug", "test", true)

	client := pushoverClient{
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

		err := expected.AddAttachment(bytes.NewReader([]byte{1, 0, 1}))
		So(err, ShouldBeNil)
		So(client.makePushoverMessage(event, trigger, [][]byte{{1, 0, 1}}, false), ShouldResemble, expected)
	})
}
