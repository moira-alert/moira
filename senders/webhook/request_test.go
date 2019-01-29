package webhook

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/moira-alert/moira"
)

var expectedPayload = `
{
  "trigger": {
    "id": "triggerID-0000000000001",
    "name": "test trigger 1",
    "description": "",
    "tags": [
      "test-tag-1"
    ]
  },
  "events": [
    {
      "metric": "metricName1",
      "value": 0,
      "timestamp": 15,
      "trigger_event": false
    },
    {
      "metric": "metricName2",
      "value": 0,
      "timestamp": 11,
      "trigger_event": false
    },
    {
      "metric": "metricName3",
      "value": 0,
      "timestamp": 31,
      "trigger_event": false
    },
    {
      "metric": "metricName4",
      "value": 0,
      "timestamp": 179,
      "trigger_event": true
    },
    {
      "metric": "metricName5",
      "value": 0,
      "timestamp": 12,
      "trigger_event": false
    }
  ],
  "contact": {
    "type": "email",
    "value": "mail1@example.com",
    "id": "ContactID-000000000000001",
    "user": "user"
  },
	"plot": ""
}
`

var (
	testContact = moira.ContactData{
		ID:    "ContactID-000000000000001",
		Type:  "email",
		Value: "mail1@example.com",
		User:  "user",
	}
	testTrigger = moira.TriggerData{
		ID:         "triggerID-0000000000001",
		Name:       "test trigger 1",
		Targets:    []string{"test.target.1"},
		WarnValue:  10,
		ErrorValue: 20,
		Tags:       []string{"test-tag-1"},
	}
	testEvents = []moira.NotificationEvent{
		{Metric: "metricName1", Timestamp: 15, IsTriggerEvent: false},
		{Metric: "metricName2", Timestamp: 11, IsTriggerEvent: false},
		{Metric: "metricName3", Timestamp: 31, IsTriggerEvent: false},
		{Metric: "metricName4", Timestamp: 179, IsTriggerEvent: true},
		{Metric: "metricName5", Timestamp: 12, IsTriggerEvent: false},
	}
	testPlot = make([]byte, 0)
)

func TestBuildRequest(t *testing.T) {
	Convey("Build request", t, func() {
		Convey("Test payload is valid", func() {
			sender := Sender{}
			events, contact, trigger, plot := testEvents, testContact, testTrigger, testPlot
			request, err := sender.buildRequest(events, contact, trigger, plot, false)
			So(err, ShouldBeNil)

			requestBodyBuff := bytes.NewBuffer([]byte{})
			err = request.Write(requestBodyBuff)
			if err != nil {
				t.Fatal(err)
			}

			fmt.Println(requestBodyBuff.String())

			actual, err := readLastLine(requestBodyBuff.String())
			if err != nil {
				t.Fatal(err)
			}

			actual, expected := prepareStrings(actual, expectedPayload, "")
			So(actual, ShouldEqual, expected)
		})

		Convey("Test url template", func() {
			events, contact, trigger, plot := testEvents, testContact, testTrigger, testPlot
			contact = moira.ContactData{
				ID:    "contactID",
				Type:  "contactType",
				Value: "contactValue",
			}
			trigger.ID = "triggerID"
			urlTemplate := "https://hostname.domain/${contact_type}/${contact_id}/${contact_value}/${trigger_id}"
			sender := Sender{url: urlTemplate}
			request, err := sender.buildRequest(events, contact, trigger, plot, false)
			So(err, ShouldBeNil)

			expected := "https://hostname.domain/contactType/contactID/contactValue/triggerID"
			actual := request.URL.String()
			So(actual, ShouldEqual, expected)
		})
	})
}

func prepareStrings(actual, expected, separator string) (string, string) {
	return strings.Join(strings.Fields(actual), separator), strings.Join(strings.Fields(expected), separator)
}

func readLastLine(longString string) (string, error) {
	reader := bytes.NewReader([]byte(longString))
	var lastLine string
	s := bufio.NewScanner(reader)
	for s.Scan() {
		lastLine = s.Text()
	}
	if err := s.Err(); err != nil {
		return "", err
	}
	return lastLine, nil
}
