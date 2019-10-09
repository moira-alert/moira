package webhook

import (
	"fmt"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/moira-alert/moira"
)

const testBadID = "!@#$"

var (
	testHost     = "https://hostname.domain"
	testTemplate = fmt.Sprintf("%s/%s/%s/%s/%s", testHost, moira.VariableTriggerID, moira.VariableContactType, moira.VariableContactID, moira.VariableContactValue)
	testContact  = moira.ContactData{
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

	testEvents = []moira.NotificationEvent{
		{Metric: "metricName1", Values: map[string]float64{"t1": 30}, Timestamp: 15, IsTriggerEvent: false, State: "OK", OldState: "ERROR"},
		{Metric: "metricName2", Values: map[string]float64{"t1": 30}, Timestamp: 11, IsTriggerEvent: false, State: "OK", OldState: "ERROR"},
		{Metric: "metricName3", Values: map[string]float64{"t1": 30}, Timestamp: 31, IsTriggerEvent: false, State: "OK", OldState: "ERROR"},
		{Metric: "metricName4", Values: map[string]float64{"t1": 30}, Timestamp: 179, IsTriggerEvent: true, State: "OK", OldState: "ERROR"},
		{Metric: "metricName5", Values: map[string]float64{"t1": 30}, Timestamp: 12, IsTriggerEvent: false, State: "OK", OldState: "ERROR"},
	}
	testPlot      = [][]byte{}
	testThrottled = false
)

const expectedStateChangePayload = `
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
      "values": {"t1":30},
      "timestamp": 15,
      "trigger_event": false,
      "state": "OK",
      "old_state": "ERROR"
    },
    {
      "metric": "metricName2",
      "values": {"t1":30},
      "timestamp": 11,
      "trigger_event": false,
      "state": "OK",
      "old_state": "ERROR"
    },
    {
      "metric": "metricName3",
      "values": {"t1":30},
      "timestamp": 31,
      "trigger_event": false,
      "state": "OK",
      "old_state": "ERROR"
    },
    {
      "metric": "metricName4",
      "values": {"t1":30},
      "timestamp": 179,
      "trigger_event": true,
      "state": "OK",
      "old_state": "ERROR"
    },
    {
      "metric": "metricName5",
      "values": {"t1":30},
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
  "plots": [],
  "throttled": false
}
`

const expectedEmptyPayload = `
{
    "trigger": {
        "id": "",
        "name": "",
        "description": "",
        "tags": []
    },
    "events": [],
    "contact": {
        "type": "",
        "value": "",
        "id": "",
        "user": ""
		},
		"plot": "",
    "plots": [],
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
		trigger: moira.TriggerData{ID: testBadID},
		contact: moira.ContactData{Type: "contactType", ID: "contactID", Value: "contactValue"},
		results: map[string]string{
			testTemplate: "https://hostname.domain/%21@%23$/contactType/contactID/contactValue",
			fmt.Sprintf("%s/%s/%s/%s", testHost, moira.VariableContactType, moira.VariableContactID, moira.VariableContactValue): "https://hostname.domain/contactType/contactID/contactValue",
		},
	},
	{
		trigger: moira.TriggerData{ID: testBadID},
		contact: moira.ContactData{Type: testBadID, ID: "contactID", Value: "contactValue"},
		results: map[string]string{
			testTemplate: "https://hostname.domain/%21@%23$/%21@%23$/contactID/contactValue",
			fmt.Sprintf("%s/%s/%s", testHost, moira.VariableContactID, moira.VariableContactValue): "https://hostname.domain/contactID/contactValue",
		},
	},
	{
		trigger: moira.TriggerData{ID: testBadID},
		contact: moira.ContactData{Type: testBadID, ID: testBadID, Value: "contactValue"},
		results: map[string]string{
			testTemplate: "https://hostname.domain/%21@%23$/%21@%23$/%21@%23$/contactValue",
			fmt.Sprintf("%s/%s", testHost, moira.VariableContactValue): "https://hostname.domain/contactValue",
		},
	},
	{
		trigger: moira.TriggerData{ID: testBadID},
		contact: moira.ContactData{Type: testBadID, ID: testBadID, Value: testBadID},
		results: map[string]string{
			testTemplate: "https://hostname.domain/%21@%23$/%21@%23$/%21@%23$/%21@%23$",
		},
	},
	{
		trigger: moira.TriggerData{},
		contact: moira.ContactData{},
		results: map[string]string{
			testHost: "https://hostname.domain",
		},
	},
}

func TestBuildRequestBody(t *testing.T) {
	Convey("Payload should be valid", t, func() {
		Convey("Trigger state change", func() {
			events, contact, trigger, plot, throttled := testEvents, testContact, testTrigger, testPlot, testThrottled
			requestBody, err := buildRequestBody(events, contact, trigger, plot, throttled)
			actual, expected := prepareStrings(string(requestBody), expectedStateChangePayload)
			So(actual, ShouldEqual, expected)
			So(err, ShouldBeNil)
		})
		Convey("Empty notification", func() {
			events, contact, trigger, plots, throttled := moira.NotificationEvents{}, moira.ContactData{}, moira.TriggerData{}, make([][]byte, 0), false
			requestBody, err := buildRequestBody(events, contact, trigger, plots, throttled)
			actual, expected := prepareStrings(string(requestBody), expectedEmptyPayload)
			So(actual, ShouldEqual, expected)
			So(actual, ShouldNotContainSubstring, "null")
			So(err, ShouldBeNil)
		})
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
