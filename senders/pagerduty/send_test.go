package pagerduty

import (
	"testing"
	"time"

	"github.com/PagerDuty/go-pagerduty"

	"github.com/moira-alert/moira"
	. "github.com/smartystreets/goconvey/convey"
)

type ImageStoreStub struct {
	moira.ImageStore
}

func (imageStore *ImageStoreStub) StoreImage(image []byte) (string, error) { return "test", nil }

func TestBuildEvent(t *testing.T) {
	location, _ := time.LoadLocation("UTC")
	sender := Sender{location: location, frontURI: "http://moira.url"}
	value := float64(97.4458331200185)

	Convey("Build pagerduty event tests", t, func() {
		event := moira.NotificationEvent{
			TriggerID: "TriggerID",
			Value:     &value,
			Timestamp: 150000000,
			Metric:    "Metric name",
			OldState:  moira.StateOK,
			State:     moira.StateNODATA,
			Message:   nil,
		}

		trigger := moira.TriggerData{
			Tags: []string{"tag1", "tag2"},
			Name: "Trigger Name",
			ID:   "TriggerID",
			Desc: "**bold text** _italics_ `code` regular",
		}

		contact := moira.ContactData{
			Value: "mock routing key",
		}
		baseExpected := pagerduty.V2Event{
			RoutingKey: contact.Value,
			Action:     "trigger",
			Payload: &pagerduty.V2Payload{
				Summary:  "NODATA [tag1][tag2]",
				Severity: "info",
				Source:   "moira",
				Details: map[string]interface{}{
					"Trigger URI": "http://moira.url/trigger/TriggerID",
					"Description": "bold text italics code regular",
				},
			},
		}

		Convey("Build pagerduty event with one moira event", func() {
			actual := sender.buildEvent(moira.NotificationEvents{event}, contact, trigger, []byte{}, false)
			expected := baseExpected
			details := map[string]interface{}{
				"Events":      "\n02:40: Metric name = 97.4458331200185 (OK to NODATA)",
				"Trigger URI": "http://moira.url/trigger/TriggerID",
				"Description": "bold text italics code regular",
			}
			expected.Payload.Details = details
			So(actual, ShouldResemble, expected)
		})

		Convey("Build pagerduty event with one moira event and plot", func() {
			sender.ImageStore = &ImageStoreStub{}
			actual := sender.buildEvent(moira.NotificationEvents{event}, contact, trigger, []byte("test"), false)
			expected := baseExpected
			details := map[string]interface{}{
				"Events":      "\n02:40: Metric name = 97.4458331200185 (OK to NODATA)",
				"Trigger URI": "http://moira.url/trigger/TriggerID",
				"Description": "bold text italics code regular",
			}
			expected.Payload.Details = details
			expected.Images = []interface{}{
				map[string]string{
					"src": "test",
					"alt": "Plot",
				},
			}
			So(actual, ShouldResemble, expected)
		})

		Convey("Build pagerduty event with one event and throttled", func() {
			actual := sender.buildEvent(moira.NotificationEvents{event}, contact, trigger, []byte{}, true)
			expected := baseExpected
			details := map[string]interface{}{
				"Events":      "\n02:40: Metric name = 97.4458331200185 (OK to NODATA)",
				"Throttled":   "Please, fix your system or tune this trigger to generate less events.",
				"Trigger URI": "http://moira.url/trigger/TriggerID",
				"Description": "bold text italics code regular",
			}
			expected.Payload.Details = details
			So(actual, ShouldResemble, expected)
		})

		Convey("Build pagerduty event with 10 events and throttled", func() {
			events := make([]moira.NotificationEvent, 0)
			for i := 0; i < 10; i++ {
				events = append(events, event)
			}
			actual := sender.buildEvent(events, contact, trigger, []byte{}, true)
			expected := baseExpected
			details := map[string]interface{}{
				"Events": `
02:40: Metric name = 97.4458331200185 (OK to NODATA)
02:40: Metric name = 97.4458331200185 (OK to NODATA)
02:40: Metric name = 97.4458331200185 (OK to NODATA)
02:40: Metric name = 97.4458331200185 (OK to NODATA)
02:40: Metric name = 97.4458331200185 (OK to NODATA)
02:40: Metric name = 97.4458331200185 (OK to NODATA)
02:40: Metric name = 97.4458331200185 (OK to NODATA)
02:40: Metric name = 97.4458331200185 (OK to NODATA)
02:40: Metric name = 97.4458331200185 (OK to NODATA)
02:40: Metric name = 97.4458331200185 (OK to NODATA)`,
				"Throttled":   "Please, fix your system or tune this trigger to generate less events.",
				"Trigger URI": "http://moira.url/trigger/TriggerID",
				"Description": "bold text italics code regular",
			}
			expected.Payload.Details = details
			So(actual, ShouldResemble, expected)
		})
	})
}
