package pagerduty

import (
	"testing"
	"time"

	"github.com/PagerDuty/go-pagerduty"

	"github.com/moira-alert/moira"
	. "github.com/smartystreets/goconvey/convey"
)

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

		Convey("Build pagerduty event with 1 moira event", func() {
			actual := sender.buildEvent(moira.NotificationEvents{event}, contact, trigger, []byte{}, false)
			expected := pagerduty.V2Event{
				RoutingKey: contact.Value,
				Action:     "trigger",
				Payload: &pagerduty.V2Payload{
					Summary:  "NODATA [tag1][tag2]",
					Severity: "info",
					Source:   "moira",
					Details: map[string]interface{}{
						"Events":      "\n02:40: Metric name = 97.4458331200185 (OK to NODATA)",
						"Trigger URI": "http://moira.url/trigger/TriggerID",
						"Description": "**bold text** _italics_ `code` regular",
					},
				},
			}
			So(actual, ShouldResemble, expected)
		})
	})
}
