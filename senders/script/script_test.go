package script

import (
	"fmt"
	"testing"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/logging/go-logging"
	. "github.com/smartystreets/goconvey/convey"
)

const testDir = "/tmp"

var (
	testTrigger = moira.TriggerData{ID: "triggerID"}
	testContact = moira.ContactData{Type: "contactType", ID: "contactID", Value: "contactValue"}
)

type execStringTestCase struct {
	template string
	expected string
}

var execStringTestCases = []execStringTestCase{
	{
		template: fmt.Sprintf("%s/%s/%s/%s/%s/script.go", testDir, moira.VariableContactType, moira.VariableContactID, moira.VariableTriggerID, moira.VariableContactValue),
		expected: "/tmp/contactType/contactID/triggerID/contactValue/script.go",
	},
	{
		template: fmt.Sprintf("%s/%s/%s/%s/script.go", testDir, moira.VariableContactID, moira.VariableTriggerID, moira.VariableContactValue),
		expected: "/tmp/contactID/triggerID/contactValue/script.go",
	},
	{
		template: fmt.Sprintf("%s/%s/%s/script.go", testDir, moira.VariableTriggerID, moira.VariableContactValue),
		expected: "/tmp/triggerID/contactValue/script.go",
	},
	{
		template: fmt.Sprintf("%s/%s/script.go", testDir, moira.VariableContactValue),
		expected: "/tmp/contactValue/script.go",
	},
	{
		template: fmt.Sprintf("%s/script.go", testDir),
		expected: "/tmp/script.go",
	},
}

func TestInit(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "debug", "test")
	Convey("Init tests", t, func() {
		sender := Sender{}
		settings := map[string]string{}
		Convey("Empty map", func() {
			err := sender.Init(settings, logger, nil, "")
			So(err, ShouldResemble, fmt.Errorf("required name for sender type script"))
			So(sender, ShouldResemble, Sender{})
		})

		settings["name"] = "script_name"
		Convey("Empty exec", func() {
			err := sender.Init(settings, logger, nil, "")
			So(err, ShouldResemble, fmt.Errorf("file  not found"))
			So(sender, ShouldResemble, Sender{})
		})

		Convey("Exec with not exists file", func() {
			settings["exec"] = "./test_file1"
			err := sender.Init(settings, logger, nil, "")
			So(err, ShouldResemble, fmt.Errorf("file ./test_file1 not found"))
			So(sender, ShouldResemble, Sender{})
		})

		Convey("Exec with exists file", func() {
			settings["exec"] = "script.go"
			err := sender.Init(settings, logger, nil, "")
			So(err, ShouldBeNil)
			So(sender, ShouldResemble, Sender{exec: "script.go", logger: logger})
		})
	})
}

func TestBuildCommandData(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "debug", "test")
	Convey("Test send events", t, func() {
		sender := Sender{exec: "script.go first second", logger: logger}
		scriptFile, args, scriptBody, err := sender.buildCommandData([]moira.NotificationEvent{{Metric: "New metric"}}, moira.ContactData{ID: "ContactID"}, moira.TriggerData{ID: "TriggerID"}, true)
		So(scriptFile, ShouldResemble, "script.go")
		So(args, ShouldResemble, []string{"first", "second"})
		So(err, ShouldBeNil)
		So(string(scriptBody), ShouldResemble, "{\n\t\"events\": [\n\t\t{\n\t\t\t\"timestamp\": 0,\n\t\t\t\"metric\": \"New metric\",\n\t\t\t\"state\": \"\",\n\t\t\t\"trigger_id\": \"\",\n\t\t\t\"old_state\": \"\"\n\t\t}\n\t],\n\t\"trigger\": {\n\t\t\"id\": \"TriggerID\",\n\t\t\"name\": \"\",\n\t\t\"desc\": \"\",\n\t\t\"targets\": null,\n\t\t\"warn_value\": 0,\n\t\t\"error_value\": 0,\n\t\t\"source_type\": \"\",\n\t\t\"__notifier_trigger_tags\": null\n\t},\n\t\"contact\": {\n\t\t\"type\": \"\",\n\t\t\"value\": \"\",\n\t\t\"id\": \"ContactID\",\n\t\t\"user\": \"\"\n\t},\n\t\"throttled\": true,\n\t\"timestamp\": 0\n}")
	})

	Convey("Test file not found", t, func() {
		sender := Sender{exec: "script1.go first second", logger: logger}
		scriptFile, args, scriptBody, err := sender.buildCommandData([]moira.NotificationEvent{{Metric: "New metric"}}, moira.ContactData{ID: "ContactID"}, moira.TriggerData{ID: "TriggerID"}, true)
		So(scriptFile, ShouldResemble, "script1.go")
		So(args, ShouldResemble, []string{"first", "second"})
		So(err, ShouldNotBeNil)
		So(scriptBody, ShouldBeEmpty)
	})

	Convey("Test exec string builder", t, func() {
		for _, testCase := range execStringTestCases {
			actual := buildExecString(testCase.template, testTrigger, testContact)
			So(actual, ShouldEqual, testCase.expected)
		}
	})
}
