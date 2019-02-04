package webhook

import (
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/moira-alert/moira"
)

const (
	testBadId    = "!@#$"
	testTemplate = "https://hostname.domain/${trigger_id}/${contact_type}/${contact_id}/${contact_value}"
)

var (
	testContact = moira.ContactData{
		ID:    "contactID",
		Type:  "contactType",
		Value: "contactValue",
		User:  "contactUser",
	}
	testTrigger = moira.TriggerData{
		ID:   "triggerID",
		Name: "triggerName for test",
		Desc: "triggerDescription",
		Tags: []string{"triggerTag1", "triggerTag2"},
	}
	testEventsValue = float64(30)
	testEvents      = []moira.NotificationEvent{
		{Metric: "metricName1", Value: &testEventsValue, Timestamp: 15, IsTriggerEvent: false, State: "OK", OldState: "ERROR"},
		{Metric: "metricName2", Value: &testEventsValue, Timestamp: 11, IsTriggerEvent: false, State: "OK", OldState: "ERROR"},
		{Metric: "metricName3", Value: &testEventsValue, Timestamp: 31, IsTriggerEvent: false, State: "OK", OldState: "ERROR"},
		{Metric: "metricName4", Value: &testEventsValue, Timestamp: 179, IsTriggerEvent: true, State: "OK", OldState: "ERROR"},
		{Metric: "metricName5", Value: &testEventsValue, Timestamp: 12, IsTriggerEvent: false, State: "OK", OldState: "ERROR"},
	}
	testPlot      = make([]byte, 0)
	testThrottled = false
)

var expectedPayload = `
{
  "trigger": {
    "id": "triggerID",
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

type requestURLTestCase struct {
	trigger moira.TriggerData
	contact moira.ContactData
	results map[string]string
}

var requestURLTestCases = []requestURLTestCase{
	{
		trigger: moira.TriggerData{ID: "triggerID"},
		contact: moira.ContactData{Type: "contactType", ID: "contactID", Value: "contactValue"},
		results: map[string]string{
			testTemplate: "https://hostname.domain/triggerID/contactType/contactID/contactValue",
		},
	},
	{
		trigger: moira.TriggerData{ID: testBadId},
		contact: moira.ContactData{Type: "contactType", ID: "contactID", Value: "contactValue"},
		results: map[string]string{
			testTemplate: "https://hostname.domain/%21@%23$/contactType/contactID/contactValue",
			"https://hostname.domain/${contact_type}/${contact_id}/${contact_value}": "https://hostname.domain/contactType/contactID/contactValue",
		},
	},
	{
		trigger: moira.TriggerData{ID: testBadId},
		contact: moira.ContactData{Type: testBadId, ID: "contactID", Value: "contactValue"},
		results: map[string]string{
			testTemplate: "https://hostname.domain/%21@%23$/%21@%23$/contactID/contactValue",
			"https://hostname.domain/${contact_id}/${contact_value}": "https://hostname.domain/contactID/contactValue",
		},
	},
	{
		trigger: moira.TriggerData{ID: testBadId},
		contact: moira.ContactData{Type: testBadId, ID: testBadId, Value: "contactValue"},
		results: map[string]string{
			testTemplate:                               "https://hostname.domain/%21@%23$/%21@%23$/%21@%23$/contactValue",
			"https://hostname.domain/${contact_value}": "https://hostname.domain/contactValue",
		},
	},
	{
		trigger: moira.TriggerData{ID: testBadId},
		contact: moira.ContactData{Type: testBadId, ID: testBadId, Value: testBadId},
		results: map[string]string{
			testTemplate: "https://hostname.domain/%21@%23$/%21@%23$/%21@%23$/%21@%23$",
		},
	},
	{
		trigger: moira.TriggerData{},
		contact: moira.ContactData{},
		results: map[string]string{
			"https://hostname.domain/": "https://hostname.domain/",
		},
	},
}

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
		for _, testCase := range requestURLTestCases {
			for k, expected := range testCase.results {
				actual := buildRequestURL(k, testCase.trigger, testCase.contact)
				So(actual, ShouldEqual, expected)
			}
		}
	})
}

func prepareStrings(actual, expected string) (string, string) {
	return strings.Join(strings.Fields(actual), ""), strings.Join(strings.Fields(expected), "")
}
