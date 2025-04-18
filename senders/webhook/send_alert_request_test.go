package webhook

import (
	"fmt"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/moira-alert/moira"
)

const (
	testBadID        = "!@#$"
	testContactType  = "testType"
	testContactValue = "testValue"
)

var (
	testHost     = "https://hostname.domain"
	testTemplate = fmt.Sprintf("%s/%s/%s/%s/%s", testHost, moira.VariableTriggerID, moira.VariableContactType, moira.VariableContactID, moira.VariableContactValue)
	testContact  = moira.ContactData{
		ID:    "contactID",
		Type:  "contactType",
		Value: "contactValue",
		User:  "contactUser",
		Team:  "contactTeam",
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
    "user": "contactUser",
    "team": "contactTeam"
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
        "user": "",
        "team": ""
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
	sender := Sender{}

	Convey("Test building default request body", t, func() {
		Convey("Trigger state change", func() {
			events, contact, trigger, plot, throttled := testEvents, testContact, testTrigger, testPlot, testThrottled
			requestBody, err := sender.buildSendAlertRequestBody(events, contact, trigger, plot, throttled)
			actual, expected := prepareStrings(string(requestBody), expectedStateChangePayload)
			So(actual, ShouldEqual, expected)
			So(err, ShouldBeNil)
		})

		Convey("Empty notification", func() {
			events, contact, trigger, plots, throttled := moira.NotificationEvents{}, moira.ContactData{}, moira.TriggerData{}, make([][]byte, 0), false
			requestBody, err := sender.buildSendAlertRequestBody(events, contact, trigger, plots, throttled)
			actual, expected := prepareStrings(string(requestBody), expectedEmptyPayload)
			So(actual, ShouldEqual, expected)
			So(actual, ShouldNotContainSubstring, "null")
			So(err, ShouldBeNil)
		})
	})

	Convey("Test building custom request body with webhook populater", t, func() {
		sender.body = "" +
			"Contact.Type: {{ .Contact.Type }}\n" +
			"Contact.Value: {{ .Contact.Value }}"

		Convey("With empty contact", func() {
			events, contact, trigger, plots, throttled := moira.NotificationEvents{}, moira.ContactData{}, moira.TriggerData{}, make([][]byte, 0), false

			requestBody, err := sender.buildSendAlertRequestBody(events, contact, trigger, plots, throttled)
			So(err, ShouldBeNil)
			So(string(requestBody), ShouldResemble, "Contact.Type: \nContact.Value:")
		})

		Convey("With only contact type", func() {
			events, contact, trigger, plots, throttled := moira.NotificationEvents{}, moira.ContactData{Type: testContactType}, moira.TriggerData{}, make([][]byte, 0), false

			requestBody, err := sender.buildSendAlertRequestBody(events, contact, trigger, plots, throttled)
			So(err, ShouldBeNil)
			So(string(requestBody), ShouldResemble, fmt.Sprintf("Contact.Type: %s\nContact.Value:", testContactType))
		})

		Convey("With only contact value", func() {
			events, contact, trigger, plots, throttled := moira.NotificationEvents{}, moira.ContactData{Value: testContactValue}, moira.TriggerData{}, make([][]byte, 0), false

			requestBody, err := sender.buildSendAlertRequestBody(events, contact, trigger, plots, throttled)
			So(err, ShouldBeNil)
			So(string(requestBody), ShouldResemble, fmt.Sprintf("Contact.Type: \nContact.Value: %s", testContactValue))
		})

		Convey("With full provided data", func() {
			events, contact, trigger, plots, throttled := moira.NotificationEvents{}, moira.ContactData{Value: testContactValue, Type: testContactType}, moira.TriggerData{}, make([][]byte, 0), false

			requestBody, err := sender.buildSendAlertRequestBody(events, contact, trigger, plots, throttled)
			So(err, ShouldBeNil)
			So(string(requestBody), ShouldResemble, fmt.Sprintf("Contact.Type: %s\nContact.Value: %s", testContactType, testContactValue))
		})
	})
}

func TestBuildRequestURL(t *testing.T) {
	Convey("URL should contain variables values", t, func() {
		for _, testCase := range requestURLTestCases {
			for k, expected := range testCase.results {
				actual := buildSendAlertRequestURL(k, testCase.trigger, testCase.contact)
				So(actual, ShouldEqual, expected)
			}
		}
	})
}

var testContactWithURL = moira.ContactData{
	ID:    "contactID",
	Type:  "contactType",
	Value: "https://test.org/moirahook",
	User:  "contactUser",
	Team:  "contactTeam",
}

func TestBuildRequestURL_FromContactValueWithURL(t *testing.T) {
	Convey("URL should contain variables values", t, func() {
		actual := buildSendAlertRequestURL("${contact_value}", testTrigger, testContactWithURL)
		So(actual, ShouldEqual, "https://test.org/moirahook")
	})
}

func prepareStrings(actual, expected string) (string, string) {
	return strings.Join(strings.Fields(actual), ""), strings.Join(strings.Fields(expected), "")
}
