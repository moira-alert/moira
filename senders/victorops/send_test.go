package victorops

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira/senders/victorops/api"

	"github.com/moira-alert/moira"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	. "github.com/smartystreets/goconvey/convey"
)

func TestBuildMessage(t *testing.T) {
	location, _ := time.LoadLocation("UTC")
	client := victoropsClient{
		location: location,
		frontURI: "http://moira.url",
	}

	Convey("Build Moira Message tests", t, func() {
		event := moira.NotificationEvent{
			TriggerID: "TriggerID",
			Values:    map[string]float64{"t1": 123},
			Timestamp: 150000000,
			Metric:    "Metric",
			OldState:  moira.StateOK,
			State:     moira.StateNODATA,
		}

		trigger := moira.TriggerData{
			Tags: []string{"tag1", "tag2"},
			Name: "Name",
			ID:   "TriggerID",
			Desc: "## test\n **test** `test` test\n",
		}

		strippedDesc := "test\n test test test\n"
		Convey("Print moira message with one event", func() {
			actual := client.buildMessage([]moira.NotificationEvent{event}, trigger, false)
			expected := strippedDesc + "\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)"
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with empty trigger", func() {
			actual := client.buildMessage([]moira.NotificationEvent{event}, moira.TriggerData{}, false)
			expected := "\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)"
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with one event and message", func() {
			var interval int64 = 24
			event.MessageEventInfo = &moira.EventInfo{Interval: &interval}
			actual := client.buildMessage([]moira.NotificationEvent{event}, trigger, false)
			expected := strippedDesc + "\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA). This metric has been in bad state for more than 24 hours - please, fix."
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with one event and throttled", func() {
			actual := client.buildMessage([]moira.NotificationEvent{event}, trigger, true)
			expected := strippedDesc + "\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\nPlease, fix your system or tune this trigger to generate less events."
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with 6 events", func() {
			actual := client.buildMessage([]moira.NotificationEvent{event, event, event, event, event, event}, trigger, false)
			expected := strippedDesc + "\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)"
			So(actual, ShouldResemble, expected)
		})

		Convey("Print moira message with empty triggerID, but with trigger name", func() {
			actual := client.buildMessage([]moira.NotificationEvent{event}, moira.TriggerData{Name: "Name"}, false)
			expected := "\n02:40 (GMT+00:00): Metric = 123 (OK to NODATA)"
			So(actual, ShouldResemble, expected)
		})
	})
}

func TestBuildCreateAlertRequest(t *testing.T) {
	location, _ := time.LoadLocation("UTC")
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	imageStore := mock_moira_alert.NewMockImageStore(mockCtrl)
	client := victoropsClient{
		location:             location,
		frontURI:             "http://moira.url",
		imageStore:           imageStore,
		imageStoreConfigured: true,
	}

	Convey("Build CreateAlertRequest tests", t, func() {
		event := moira.NotificationEvent{
			TriggerID: "TriggerID",
			Values:    map[string]float64{"t1": 123},
			Timestamp: 150000000,
			Metric:    "Metric",
			OldState:  moira.StateOK,
			State:     moira.StateNODATA,
		}

		trigger := moira.TriggerData{
			Tags: []string{"tag1", "tag2"},
			Name: "Name",
			ID:   "TriggerID",
		}

		Convey("Build CreateAlertRequest with one moira event and plot", func() {
			imageStore.EXPECT().StoreImage([]byte("test")).Return("test", nil)
			actual := client.buildCreateAlertRequest(moira.NotificationEvents{event}, trigger, false, [][]byte{[]byte("test")}, 150000000)
			expected := api.CreateAlertRequest{
				MessageType:       api.Warning,
				StateMessage:      client.buildMessage(moira.NotificationEvents{event}, trigger, false),
				EntityID:          trigger.ID,
				Timestamp:         150000000,
				StateStartTime:    event.Timestamp,
				TriggerURL:        "http://moira.url/trigger/TriggerID",
				ImageURL:          "test",
				MonitoringTool:    "Moira",
				EntityDisplayName: client.buildTitle(moira.NotificationEvents{event}, trigger, false),
			}

			So(actual, ShouldResemble, expected)
		})
	})
}

func TestBuildTitle(t *testing.T) {
	client := victoropsClient{}

	Convey("Build title test", t, func() {
		events := moira.NotificationEvents{
			moira.NotificationEvent{
				TriggerID: "TriggerID",
				Values:    map[string]float64{"t1": 123},
				Timestamp: 150000000,
				Metric:    "Metric",
				OldState:  moira.StateOK,
				State:     moira.StateNODATA,
				Message:   nil,
			},
			moira.NotificationEvent{
				TriggerID: "TriggerID",
				Values:    map[string]float64{"t1": 15},
				Timestamp: 150000000,
				Metric:    "Metric",
				OldState:  moira.StateNODATA,
				State:     moira.StateOK,
				Message:   nil,
			},
		}

		trigger := moira.TriggerData{
			Tags: []string{"tag1", "tag2"},
			Name: "Name",
			ID:   "TriggerID",
		}

		Convey("Build title without throttling", func() {
			actual := client.buildTitle(events, trigger, false)
			expected := "NODATA Name [tag1][tag2]\n"
			So(actual, ShouldResemble, expected)
		})

		Convey("Build title when throttling", func() {
			actual := client.buildTitle(events, trigger, true)
			expected := "OK Name [tag1][tag2]\n"
			So(actual, ShouldResemble, expected)
		})
	})
}
