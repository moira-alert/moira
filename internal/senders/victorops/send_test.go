package victorops

import (
	"testing"
	"time"

	moira2 "github.com/moira-alert/moira/internal/moira"

	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira/internal/senders/victorops/api"

	mock_moira_alert "github.com/moira-alert/moira/internal/mock/moira-alert"
	. "github.com/smartystreets/goconvey/convey"
)

func TestBuildMessage(t *testing.T) {
	location, _ := time.LoadLocation("UTC")
	sender := Sender{location: location, frontURI: "http://moira.url"}
	value := float64(123)

	Convey("Build Moira Message tests", t, func() {
		event := moira2.NotificationEvent{
			TriggerID: "TriggerID",
			Value:     &value,
			Timestamp: 150000000,
			Metric:    "Metric",
			OldState:  moira2.StateOK,
			State:     moira2.StateNODATA,
		}

		trigger := moira2.TriggerData{
			Tags: []string{"tag1", "tag2"},
			Name: "Name",
			ID:   "TriggerID",
			Desc: "## test\n **test** `test` test\n",
		}

		strippedDesc := "test\n test test test\n"
		Convey("Print moira message with one event", func() {
			actual := sender.buildMessage([]moira2.NotificationEvent{event}, trigger, false)
			expected := strippedDesc + "\n02:40: Metric = 123 (OK to NODATA)"
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with empty trigger", func() {
			actual := sender.buildMessage([]moira2.NotificationEvent{event}, moira2.TriggerData{}, false)
			expected := "\n02:40: Metric = 123 (OK to NODATA)"
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with one event and message", func() {
			var interval int64 = 24
			event.MessageEventInfo = &moira2.EventInfo{Interval: &interval}
			actual := sender.buildMessage([]moira2.NotificationEvent{event}, trigger, false)
			expected := strippedDesc + "\n02:40: Metric = 123 (OK to NODATA). This metric has been in bad state for more than 24 hours - please, fix."
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with one event and throttled", func() {
			actual := sender.buildMessage([]moira2.NotificationEvent{event}, trigger, true)
			expected := strippedDesc + "\n02:40: Metric = 123 (OK to NODATA)\nPlease, fix your system or tune this trigger to generate less events."
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with 6 events", func() {
			actual := sender.buildMessage([]moira2.NotificationEvent{event, event, event, event, event, event}, trigger, false)
			expected := strippedDesc + "\n02:40: Metric = 123 (OK to NODATA)\n02:40: Metric = 123 (OK to NODATA)\n02:40: Metric = 123 (OK to NODATA)\n02:40: Metric = 123 (OK to NODATA)\n02:40: Metric = 123 (OK to NODATA)\n02:40: Metric = 123 (OK to NODATA)"
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with empty triggerID, but with trigger name", func() {
			actual := sender.buildMessage([]moira2.NotificationEvent{event}, moira2.TriggerData{Name: "Name"}, false)
			expected := "\n02:40: Metric = 123 (OK to NODATA)"
			So(actual, ShouldResemble, expected)
		})

	})
}

func TestBuildCreateAlertRequest(t *testing.T) {
	location, _ := time.LoadLocation("UTC")
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	imageStore := mock_moira_alert.NewMockImageStore(mockCtrl)
	sender := Sender{location: location, frontURI: "http://moira.url", imageStore: imageStore, imageStoreConfigured: true}
	value := float64(123)

	Convey("Build CreateAlertRequest tests", t, func() {
		event := moira2.NotificationEvent{
			TriggerID: "TriggerID",
			Value:     &value,
			Timestamp: 150000000,
			Metric:    "Metric",
			OldState:  moira2.StateOK,
			State:     moira2.StateNODATA,
		}

		trigger := moira2.TriggerData{
			Tags: []string{"tag1", "tag2"},
			Name: "Name",
			ID:   "TriggerID",
		}

		Convey("Build CreateAlertRequest with one moira event and plot", func() {
			imageStore.EXPECT().StoreImage([]byte("test")).Return("test", nil)
			actual := sender.buildCreateAlertRequest(moira2.NotificationEvents{event}, trigger, false, []byte("test"), 150000000)
			expected := api.CreateAlertRequest{
				MessageType:       api.Warning,
				StateMessage:      sender.buildMessage(moira2.NotificationEvents{event}, trigger, false),
				EntityID:          trigger.ID,
				Timestamp:         150000000,
				StateStartTime:    event.Timestamp,
				TriggerURL:        "http://moira.url/trigger/TriggerID",
				ImageURL:          "test",
				MonitoringTool:    "Moira",
				EntityDisplayName: sender.buildTitle(moira2.NotificationEvents{event}, trigger),
			}
			So(actual, ShouldResemble, expected)
		})
	})
}

func TestBuildTitle(t *testing.T) {
	sender := Sender{}
	value := float64(123)

	Convey("Build title test", t, func() {
		event := moira2.NotificationEvent{
			TriggerID: "TriggerID",
			Value:     &value,
			Timestamp: 150000000,
			Metric:    "Metric",
			OldState:  moira2.StateOK,
			State:     moira2.StateNODATA,
			Message:   nil,
		}

		trigger := moira2.TriggerData{
			Tags: []string{"tag1", "tag2"},
			Name: "Name",
			ID:   "TriggerID",
		}

		Convey("Build title", func() {
			actual := sender.buildTitle(moira2.NotificationEvents{event}, trigger)
			expected := "NODATA Name [tag1][tag2]\n"
			So(actual, ShouldResemble, expected)
		})
	})
}
