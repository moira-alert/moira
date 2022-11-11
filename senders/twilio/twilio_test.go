package twilio

import (
	"fmt"
	"testing"
	"time"

	twilio "github.com/carlosdp/twiliogo"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	. "github.com/smartystreets/goconvey/convey"
)

func TestNewSender(t *testing.T) {
	Convey("Tests init twilio sender", t, func() {
		logger, _ := logging.ConfigureLog("stdout", "debug", "test", true)
		location, _ := time.LoadLocation("UTC")
		settings := map[string]string{}
		Convey("no api asid", func() {
			sender, err := NewSender(settings, logger, nil)
			So(err, ShouldResemble, fmt.Errorf("can not read [%s] api_sid param from config", ""))
			So(sender, ShouldBeNil)
		})

		settings["api_asid"] = "123"

		Convey("no api authtoken", func() {
			sender, err := NewSender(settings, logger, nil)
			So(err, ShouldResemble, fmt.Errorf("can not read [%s] api_authtoken param from config", ""))
			So(sender, ShouldBeNil)
		})

		settings["api_authtoken"] = "321"

		Convey("no api fromphone", func() {
			sender, err := NewSender(settings, logger, nil)
			So(err, ShouldResemble, fmt.Errorf("can not read [%s] api_fromphone param from config", ""))
			So(sender, ShouldBeNil)
		})

		settings["api_fromphone"] = "12345678989"

		Convey("no api type", func() {
			sender, err := NewSender(settings, logger, nil)
			So(err, ShouldResemble, fmt.Errorf("wrong twilio type: %s", ""))
			So(sender, ShouldBeNil)
		})

		Convey("config sms", func() {
			settings["type"] = "twilio sms"
			sender, err := NewSender(settings, logger, location)
			So(err, ShouldBeNil)
			So(sender, ShouldResemble, &Sender{sender: &twilioSenderSms{
				twilioSender{
					client:       twilio.NewClient("123", "321"),
					APIFromPhone: "12345678989",
					logger:       logger,
					location:     location,
				},
			}})
		})

		Convey("config voice", func() {
			settings["type"] = "twilio voice"
			Convey("no voice url", func() {
				sender, err := NewSender(settings, logger, nil)
				So(err, ShouldResemble, fmt.Errorf("can not read [%s] voiceurl param from config", "twilio voice"))
				So(sender, ShouldBeNil)
			})

			Convey("has voice url", func() {
				settings["voiceurl"] = "url here"
				Convey("append_message == true", func() {
					settings["append_message"] = "true"
					sender, err := NewSender(settings, logger, location)
					So(err, ShouldBeNil)
					So(sender, ShouldResemble, &Sender{sender: &twilioSenderVoice{
						twilioSender: twilioSender{
							client:       twilio.NewClient("123", "321"),
							APIFromPhone: "12345678989",
							logger:       logger,
							location:     location,
						},
						voiceURL:      "url here",
						appendMessage: true,
					}})
				})

				Convey("append_message is something another string", func() {
					settings["append_message"] = "something another string"
					sender, err := NewSender(settings, logger, location)
					So(err, ShouldBeNil)
					So(sender, ShouldResemble, &Sender{sender: &twilioSenderVoice{
						twilioSender: twilioSender{
							client:       twilio.NewClient("123", "321"),
							APIFromPhone: "12345678989",
							logger:       logger,
							location:     location,
						},
						voiceURL:      "url here",
						appendMessage: false,
					}})
				})
			})
		})
	})
}
