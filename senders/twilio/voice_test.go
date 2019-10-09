package twilio

import (
	"time"

	twilio "github.com/carlosdp/twiliogo"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/logging/go-logging"
	. "github.com/smartystreets/goconvey/convey"

	"testing"
)

func TestTwilioSenderVoice_SendEvents(t *testing.T) {
	logger, _ := logging.ConfigureLog("stdout", "debug", "test")
	location, _ := time.LoadLocation("UTC")
	sender := twilioSenderVoice{
		twilioSender: twilioSender{
			client:       twilio.NewClient("123", "321"),
			APIFromPhone: "12345678989",
			logger:       logger,
			location:     location,
		},
		voiceURL:      "url here",
		appendMessage: false,
	}

	Convey("just send", t, func() {
		err := sender.SendEvents(moira.NotificationEvents{}, moira.ContactData{}, moira.TriggerData{}, [][]byte{}, true)
		So(err, ShouldNotBeNil)
	})
}

func TestBuildVoiceURL(t *testing.T) {
	sender := twilioSenderVoice{
		twilioSender: twilioSender{
			client:       twilio.NewClient("123", "321"),
			APIFromPhone: "12345678989",
		},
		voiceURL:      "url here",
		appendMessage: false,
	}
	Convey("append message is false", t, func() {
		So(sender.buildVoiceURL(moira.TriggerData{Name: "Name"}), ShouldResemble, "url here")
	})

	Convey("append message is true", t, func() {
		sender.appendMessage = true
		So(sender.buildVoiceURL(moira.TriggerData{Name: "Name"}), ShouldResemble, "url hereHi%21+This+is+a+notification+for+Moira+trigger+Name.+Please%2C+visit+Moira+web+interface+for+details.")
	})

	Convey("twimlets echo is true", t, func() {
		sender.twimletsEcho = true
		So(sender.buildVoiceURL(moira.TriggerData{Name: "Name"}), ShouldResemble,
			"https://twimlets.com/echo?Twiml=%3CResponse%3E%3CSay%3EHi%21+This+is+a+notification+for+Moira+trigger+Name.+Please%2C+visit+Moira+web+interface+for+details.%3C%2FSay%3E%3C%2FResponse%3E")
	})
}
