package mail

import (
	"bytes"
	"fmt"
	"html/template"
	"testing"
	"time"

	moira2 "github.com/moira-alert/moira/internal/moira"

	"github.com/moira-alert/moira/internal/logging/go-logging"
	. "github.com/smartystreets/goconvey/convey"
)

func TestMakeMessage(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "debug", "test")
	contact := moira2.ContactData{
		ID:    "ContactID-000000000000001",
		Type:  "email",
		Value: "mail1@example.com",
	}

	trigger := moira2.TriggerData{
		ID:         "triggerID-0000000000001",
		Name:       "test trigger 1",
		Targets:    []string{"test.target.1"},
		WarnValue:  10,
		ErrorValue: 20,
		Tags:       []string{"test-tag-1"},
		Desc: `# header 1
some text **bold text**
## header 2
some other text _italics text_`,
	}

	location, _ := time.LoadLocation("UTC")
	templateName := "mail"

	sender := Sender{
		FrontURI:     "http://localhost",
		From:         "test@notifier",
		SMTPHost:     "localhost",
		SMTPPort:     25,
		Template:     template.Must(template.New(templateName).Parse(defaultTemplate)),
		TemplateName: templateName,
		location:     location,
		logger:       logger,
	}

	Convey("Make message", t, func() {
		message := sender.makeMessage(generateTestEvents(10, trigger.ID), contact, trigger, []byte{1, 0, 1}, true)
		So(message.GetHeader("From")[0], ShouldEqual, sender.From)
		So(message.GetHeader("To")[0], ShouldEqual, contact.Value)

		messageStr := new(bytes.Buffer)
		_, err := message.WriteTo(messageStr)
		So(err, ShouldBeNil)
		So(messageStr.String(), ShouldContainSubstring, "http://localhost/trigger/triggerID-0000000000001")
		So(messageStr.String(), ShouldContainSubstring, "<em>italics text</em>")
		So(messageStr.String(), ShouldContainSubstring, "<strong>bold text</strong>")
		//fmt.Println(messageStr.String())

	})
}

func generateTestEvents(n int, subscriptionID string) []moira2.NotificationEvent {
	events := make([]moira2.NotificationEvent, 0, n)
	for i := 0; i < n; i++ {
		event := moira2.NotificationEvent{
			Metric:         fmt.Sprintf("Metric number #%d", i),
			SubscriptionID: &subscriptionID,
			State:          moira2.StateTEST,
		}
		events = append(events, event)
	}
	return events
}

func TestEmptyTriggerID(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "debug", "test")
	contact := moira2.ContactData{
		ID:    "ContactID-000000000000001",
		Type:  "email",
		Value: "mail1@example.com",
	}

	trigger := moira2.TriggerData{
		ID:         "",
		Name:       "test trigger 2",
		Targets:    []string{"test.target.1"},
		WarnValue:  10,
		ErrorValue: 20,
		Tags:       []string{"test-tag-1"},
	}

	location, _ := time.LoadLocation("UTC")
	templateName := "mail"

	sender := Sender{
		FrontURI:     "http://localhost",
		From:         "test@notifier",
		SMTPHost:     "localhost",
		SMTPPort:     25,
		Template:     template.Must(template.New(templateName).Parse(defaultTemplate)),
		TemplateName: templateName,
		location:     location,
		logger:       logger,
	}

	Convey("Make message", t, func() {
		message := sender.makeMessage(generateTestEvents(10, trigger.ID), contact, trigger, []byte{1, 0, 1}, true)
		So(message.GetHeader("From")[0], ShouldEqual, sender.From)
		So(message.GetHeader("To")[0], ShouldEqual, contact.Value)
		messageStr := new(bytes.Buffer)
		_, err := message.WriteTo(messageStr)
		So(err, ShouldBeNil)
		So(messageStr.String(), ShouldNotContainSubstring, "http://localhost/trigger/")
		So(messageStr.String(), ShouldNotContainSubstring, "<p><a href=3D\"\"></a></p>")
		//fmt.Println(messageStr.String())
	})
}
