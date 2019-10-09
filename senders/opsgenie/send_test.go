package opsgenie

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira"
	logging "github.com/moira-alert/moira/logging/go-logging"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	"github.com/opsgenie/opsgenie-go-sdk-v2/alert"
	. "github.com/smartystreets/goconvey/convey"
)

func TestGetPushoverPriority(t *testing.T) {
	sender := Sender{}
	Convey("All events has OK state", t, func() {
		priority := sender.getMessagePriority([]moira.NotificationEvent{{State: moira.StateOK}, {State: moira.StateOK}, {State: moira.StateOK}})
		So(priority, ShouldResemble, alert.P5)
	})

	Convey("One of events has WARN state", t, func() {
		priority := sender.getMessagePriority([]moira.NotificationEvent{{State: moira.StateOK}, {State: moira.StateWARN}, {State: moira.StateOK}})
		So(priority, ShouldResemble, alert.P3)
	})

	Convey("One of events has NODATA state", t, func() {
		priority := sender.getMessagePriority([]moira.NotificationEvent{{State: moira.StateOK}, {State: moira.StateNODATA}, {State: moira.StateOK}})
		So(priority, ShouldResemble, alert.P3)
	})

	Convey("One of events has ERROR state", t, func() {
		priority := sender.getMessagePriority([]moira.NotificationEvent{{State: moira.StateOK}, {State: moira.StateERROR}, {State: moira.StateOK}})
		So(priority, ShouldResemble, alert.P1)
	})

	Convey("One of events has EXCEPTION state", t, func() {
		priority := sender.getMessagePriority([]moira.NotificationEvent{{State: moira.StateOK}, {State: moira.StateEXCEPTION}, {State: moira.StateOK}})
		So(priority, ShouldResemble, alert.P1)
	})

	Convey("Events has WARN and ERROR states", t, func() {
		priority := sender.getMessagePriority([]moira.NotificationEvent{{State: moira.StateOK}, {State: moira.StateWARN}, {State: moira.StateERROR}})
		So(priority, ShouldResemble, alert.P1)
	})

	Convey("Events has ERROR and WARN states", t, func() {
		priority := sender.getMessagePriority([]moira.NotificationEvent{{State: moira.StateOK}, {State: moira.StateERROR}, {State: moira.StateWARN}})
		So(priority, ShouldResemble, alert.P1)
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

		trigger := moira.TriggerData{
			Desc: `## header
**bold text** _italics_` + "\n```code```",
		}

		Convey("Print moira message with one event", func() {
			actual := sender.buildMessage([]moira.NotificationEvent{event}, false, moira.TriggerData{})
			expected := "02:40: Metric = t1:123 (OK to NODATA)\n"
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with one event and desc", func() {
			actual := sender.buildMessage([]moira.NotificationEvent{event}, false, trigger)
			expected := "<h2>header</h2>\n\n<p><strong>bold text</strong> <em>italics</em>\n<code>code</code></p>\n" + "02:40: Metric = t1:123 (OK to NODATA)\n"
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with one event and message", func() {
			var interval int64 = 24
			event.MessageEventInfo = &moira.EventInfo{Interval: &interval}
			actual := sender.buildMessage([]moira.NotificationEvent{event}, false, moira.TriggerData{})
			expected := "02:40: Metric = t1:123 (OK to NODATA). This metric has been in bad state for more than 24 hours - please, fix.\n"
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with one event and throttled", func() {
			actual := sender.buildMessage([]moira.NotificationEvent{event}, true, moira.TriggerData{})
			expected := `02:40: Metric = t1:123 (OK to NODATA)

Please, fix your system or tune this trigger to generate less events.`
			So(actual, ShouldResemble, expected)
		})
	})
}

func TestBuildTitle(t *testing.T) {
	sender := Sender{}
	Convey("Build title with three events with max ERROR state and two tags", t, func() {
		title := sender.buildTitle([]moira.NotificationEvent{{State: moira.StateERROR}, {State: moira.StateWARN}, {State: moira.StateWARN}, {State: moira.StateOK}}, moira.TriggerData{Tags: []string{"tag1", "tag2"}, Name: "Name"})
		So(title, ShouldResemble, "ERROR Name [tag1][tag2] (4)")
	})
	Convey("Build title with three events with max ERROR state empty trigger", t, func() {
		title := sender.buildTitle([]moira.NotificationEvent{{State: moira.StateERROR}, {State: moira.StateWARN}, {State: moira.StateWARN}, {State: moira.StateOK}}, moira.TriggerData{})
		So(title, ShouldResemble, "ERROR   (4)")
	})
	Convey("Build title that exceeds the title limit", t, func() {
		var reallyLongTag string
		for i := 0; i < 30; i++ {
			reallyLongTag = reallyLongTag + "randomstring"
		}
		title := sender.buildTitle([]moira.NotificationEvent{{State: moira.StateERROR}, {State: moira.StateWARN}, {State: moira.StateWARN}, {State: moira.StateOK}}, moira.TriggerData{Tags: []string{"tag1", "tag2", "tag3", reallyLongTag, "tag4"}, Name: "Name"})
		So(title, ShouldResemble, "ERROR Name [tag1][tag2][tag3].... (4)")
	})
}

func TestMakeCreateAlertRequest(t *testing.T) {
	location, _ := time.LoadLocation("UTC")
	logger, _ := logging.ConfigureLog("stdout", "debug", "test")
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	imageStore := mock_moira_alert.NewMockImageStore(mockCtrl)

	sender := Sender{
		frontURI:             "https://my-moira.com",
		location:             location,
		logger:               logger,
		imageStoreConfigured: true,
		imageStore:           imageStore,
	}
	imageStore.EXPECT().StoreImage([]byte(`test`)).Return("testlink", nil)
	Convey("Build CreateAlertRequest", t, func() {
		event := []moira.NotificationEvent{{
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
		contact := moira.ContactData{
			Value: "123",
		}
		actual := sender.makeCreateAlertRequest(event, contact, trigger, [][]byte{[]byte(`test`)}, false)
		expected := &alert.CreateAlertRequest{
			Message:     sender.buildTitle(event, trigger),
			Description: sender.buildMessage(event, false, trigger),
			Alias:       "SomeID",
			Responders: []alert.Responder{
				{Type: alert.EscalationResponder, Name: contact.Value},
			},
			Tags:     trigger.Tags,
			Source:   "Moira",
			Priority: sender.getMessagePriority(event),
			Details: map[string]string{
				"image_url": "testlink",
			},
		}
		So(actual, ShouldResemble, expected)
	})
}
