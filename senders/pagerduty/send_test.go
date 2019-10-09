package pagerduty

import (
	"testing"
	"time"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/golang/mock/gomock"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"

	"github.com/moira-alert/moira"
	. "github.com/smartystreets/goconvey/convey"
)

func TestBuildEvent(t *testing.T) {
	location, _ := time.LoadLocation("UTC")
	sender := Sender{location: location, frontURI: "http://moira.url"}
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	imageStore := mock_moira_alert.NewMockImageStore(mockCtrl)

	Convey("Build pagerduty event tests", t, func() {
		event := moira.NotificationEvent{
			TriggerID: "TriggerID",
			Values:    map[string]float64{"t1": 97.4458331200185},
			Timestamp: 150000000,
			Metric:    "Metric name",
			OldState:  moira.StateOK,
			State:     moira.StateNODATA,
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
				Summary:   "NODATA Trigger Name [tag1][tag2]",
				Severity:  "warning",
				Source:    "moira",
				Timestamp: "1974-10-03T02:40:00Z",
				Details: map[string]interface{}{
					"Trigger URI":  "http://moira.url/trigger/TriggerID",
					"Trigger Name": "Trigger Name",
					"Description":  "bold text italics code regular",
				},
			},
		}

		Convey("Build pagerduty event with one moira event", func() {
			actual := sender.buildEvent(moira.NotificationEvents{event}, contact, trigger, [][]byte{}, false)
			expected := baseExpected
			details := map[string]interface{}{
				"Events":       "\n02:40: Metric name = t1:97.4458331200185 (OK to NODATA)",
				"Trigger URI":  "http://moira.url/trigger/TriggerID",
				"Trigger Name": "Trigger Name",
				"Description":  "bold text italics code regular",
			}
			expected.Payload.Details = details
			So(actual, ShouldResemble, expected)
		})

		Convey("Build pagerduty event with one moira event and plot", func() {
			imageStore.EXPECT().StoreImage([]byte("test")).Return("test", nil)
			sender.imageStore = imageStore
			sender.imageStoreConfigured = true
			actual := sender.buildEvent(moira.NotificationEvents{event}, contact, trigger, [][]byte{[]byte("test")}, false)
			expected := baseExpected
			details := map[string]interface{}{
				"Events":       "\n02:40: Metric name = t1:97.4458331200185 (OK to NODATA)",
				"Trigger URI":  "http://moira.url/trigger/TriggerID",
				"Trigger Name": "Trigger Name",
				"Description":  "bold text italics code regular",
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
			actual := sender.buildEvent(moira.NotificationEvents{event}, contact, trigger, [][]byte{}, true)
			expected := baseExpected
			details := map[string]interface{}{
				"Events":       "\n02:40: Metric name = t1:97.4458331200185 (OK to NODATA)",
				"Message":      "Please, fix your system or tune this trigger to generate less events.",
				"Trigger URI":  "http://moira.url/trigger/TriggerID",
				"Description":  "bold text italics code regular",
				"Trigger Name": "Trigger Name",
			}
			expected.Payload.Details = details
			So(actual, ShouldResemble, expected)
		})

		Convey("Build pagerduty event with 10 events and throttled", func() {
			events := make([]moira.NotificationEvent, 0)
			for i := 0; i < 10; i++ {
				events = append(events, event)
			}
			actual := sender.buildEvent(events, contact, trigger, [][]byte{}, true)
			expected := baseExpected
			details := map[string]interface{}{
				"Events": `
02:40: Metric name = t1:97.4458331200185 (OK to NODATA)
02:40: Metric name = t1:97.4458331200185 (OK to NODATA)
02:40: Metric name = t1:97.4458331200185 (OK to NODATA)
02:40: Metric name = t1:97.4458331200185 (OK to NODATA)
02:40: Metric name = t1:97.4458331200185 (OK to NODATA)
02:40: Metric name = t1:97.4458331200185 (OK to NODATA)
02:40: Metric name = t1:97.4458331200185 (OK to NODATA)
02:40: Metric name = t1:97.4458331200185 (OK to NODATA)
02:40: Metric name = t1:97.4458331200185 (OK to NODATA)
02:40: Metric name = t1:97.4458331200185 (OK to NODATA)`,
				"Message":      "Please, fix your system or tune this trigger to generate less events.",
				"Trigger URI":  "http://moira.url/trigger/TriggerID",
				"Description":  "bold text italics code regular",
				"Trigger Name": "Trigger Name",
			}
			expected.Payload.Details = details
			So(actual, ShouldResemble, expected)
		})
	})
}
