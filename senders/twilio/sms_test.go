package twilio

import (
	"testing"
	"time"

	twilio "github.com/carlosdp/twiliogo"
	"github.com/moira-alert/moira"
	logging "github.com/moira-alert/moira/logging/go-logging"
	. "github.com/smartystreets/goconvey/convey"
)

func TestBuildMoiraMessage(t *testing.T) {
	location, _ := time.LoadLocation("UTC")
	sender := twilioSenderSms{
		twilioSender: twilioSender{
			location: location,
		}}

	Convey("Build Moira Message tests", t, func() {
		event := moira.NotificationEvent{
			Values:    map[string]float64{"t1": 123},
			Timestamp: 150000000,
			Metric:    "Metric",
			OldState:  moira.StateOK,
			State:     moira.StateNODATA,
		}

		Convey("Print moira message with one event", func() {
			actual := sender.buildMessage([]moira.NotificationEvent{event}, moira.TriggerData{Name: "Name", Tags: []string{"tag1"}}, false)
			expected := "NODATA Name [tag1] (1)\n\n02:40: Metric = t1:123 (OK to NODATA)"
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with one event and message", func() {
			var interval int64 = 24
			event.MessageEventInfo = &moira.EventInfo{Interval: &interval}
			actual := sender.buildMessage([]moira.NotificationEvent{event}, moira.TriggerData{Name: "Name", Tags: []string{"tag1"}}, false)
			expected := "NODATA Name [tag1] (1)\n\n02:40: Metric = t1:123 (OK to NODATA). This metric has been in bad state for more than 24 hours - please, fix."
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with one event and throttled", func() {
			actual := sender.buildMessage([]moira.NotificationEvent{event}, moira.TriggerData{Name: "Name", Tags: []string{"tag1"}}, true)
			expected := `NODATA Name [tag1] (1)

02:40: Metric = t1:123 (OK to NODATA)

Please, fix your system or tune this trigger to generate less events.`
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with 6 events", func() {
			actual := sender.buildMessage([]moira.NotificationEvent{event, event, event, event, event, event}, moira.TriggerData{Name: "Name", Tags: []string{"tag1"}}, false)
			expected := `NODATA Name [tag1] (6)

02:40: Metric = t1:123 (OK to NODATA)
02:40: Metric = t1:123 (OK to NODATA)
02:40: Metric = t1:123 (OK to NODATA)
02:40: Metric = t1:123 (OK to NODATA)
02:40: Metric = t1:123 (OK to NODATA)

...and 1 more events.`
			So(actual, ShouldResemble, expected)
		})
	})
}

func TestTwilioSenderSms_SendEvents(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "debug", "test")
	location, _ := time.LoadLocation("UTC")
	sender := twilioSenderSms{
		twilioSender: twilioSender{
			client:       twilio.NewClient("123", "321"),
			APIFromPhone: "12345678989",
			logger:       logger,
			location:     location,
		},
	}

	event := moira.NotificationEvent{
		Values:    map[string]float64{"t1": 123},
		Timestamp: 150000000,
		Metric:    "Metric",
		OldState:  moira.StateOK,
		State:     moira.StateNODATA,
	}

	Convey("just send", t, func() {
		err := sender.SendEvents([]moira.NotificationEvent{event}, moira.ContactData{}, moira.TriggerData{Name: "Name", Tags: []string{"tag1"}}, [][]byte{}, true)
		So(err, ShouldNotBeNil)
	})
}
