package webhook

import (
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/moira-alert/moira"
)

var (
	testContact = moira.ContactData{
		ID:    "contactID",
		Type:  "contactType",
		Value: "contactValue",
		User:  "contactUser",
	}
	testTrigger = moira.TriggerData{
		ID:         "!@#$",
		Name:       "triggerName for test",
		Desc:       "triggerDescription",
		Tags:       []string{"triggerTag1", "triggerTag2"},
	}
	testEventsValue = float64(30)
	testEvents      = []moira.NotificationEvent{
		{Metric: "metricName1", Value: &testEventsValue, Timestamp: 15, IsTriggerEvent: false, State: "OK", OldState: "ERROR"},
		{Metric: "metricName2", Value: &testEventsValue, Timestamp: 11, IsTriggerEvent: false, State: "OK", OldState: "ERROR"},
		{Metric: "metricName3", Value: &testEventsValue, Timestamp: 31, IsTriggerEvent: false, State: "OK", OldState: "ERROR"},
		{Metric: "metricName4", Value: &testEventsValue, Timestamp: 179, IsTriggerEvent: true, State: "OK", OldState: "ERROR"},
		{Metric: "metricName5", Value: &testEventsValue, Timestamp: 12, IsTriggerEvent: false, State: "OK", OldState: "ERROR"},
	}
	testPlot = make([]byte, 0)
	testThrottled = false
)

var expectedPayload = `
{
  "trigger": {
    "id": "!@#$",
    "name": "triggerName for test",
    "description": "triggerDescription",
    "tags": [
      "triggerTag1",
      "triggerTag2"
    ]
  },
  "events": [
    {
      "metric": "metricName1",
      "value": 30,
      "timestamp": 15,
      "trigger_event": false,
      "state": "OK",
      "old_state": "ERROR"
    },
    {
      "metric": "metricName2",
      "value": 30,
      "timestamp": 11,
      "trigger_event": false,
      "state": "OK",
      "old_state": "ERROR"
    },
    {
      "metric": "metricName3",
      "value": 30,
      "timestamp": 31,
      "trigger_event": false,
      "state": "OK",
      "old_state": "ERROR"
    },
    {
      "metric": "metricName4",
      "value": 30,
      "timestamp": 179,
      "trigger_event": true,
      "state": "OK",
      "old_state": "ERROR"
    },
    {
      "metric": "metricName5",
      "value": 30,
      "timestamp": 12,
      "trigger_event": false,
      "state": "OK",
      "old_state": "ERROR"
    }
  ],
  "contact": {
    "type": "contactType",
    "value": "contactValue",
    "id": "contactID",
    "user": "contactUser"
  },
  "plot": "",
  "throttled": false
}
`

func TestBuildRequestBody(t *testing.T) {
	Convey("Payload should be valid", t, func() {
		requestBody, err := buildRequestBody(testEvents, testContact, testTrigger, testPlot, testThrottled)
		actual, expected := prepareStrings(string(requestBody), expectedPayload)
		So(actual, ShouldEqual, expected)
		So(err, ShouldBeNil)
	})
}

func TestBuildRequestURL(t *testing.T) {
	Convey("URL should contain variables values", t, func() {
		template := "https://hostname.domain/${contact_type}/${contact_id}/${contact_value}/${trigger_id}"
		expected := "https://hostname.domain/contactType/contactID/contactValue/%21@%23$"
		actual := buildRequestURL(template, testTrigger, testContact)
		So(actual, ShouldEqual, expected)
	})
}

func prepareStrings(actual, expected string) (string, string) {
	return strings.Join(strings.Fields(actual), ""), strings.Join(strings.Fields(expected), "")
}
