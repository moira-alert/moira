package twilio

import (
	"fmt"
	"testing"
	"time"

	twilio "github.com/carlosdp/twiliogo"
	"github.com/moira-alert/moira/logging/go-logging"
	. "github.com/smartystreets/goconvey/convey"
)

func TestInit(t *testing.T) {
	Convey("Tests init twilio sender", t, func(c C) {
		sender := Sender{}
		logger, _ := logging.ConfigureLog("stdout", "debug", "test")
		location, _ := time.LoadLocation("UTC")
		settings := map[string]string{}
		Convey("no api asid", t, func(c C) {
			err := sender.Init(settings, logger, nil, "15:04")
			c.So(err, ShouldResemble, fmt.Errorf("can not read [%s] api_sid param from config", ""))
			c.So(sender, ShouldResemble, Sender{})
		})

		settings["api_asid"] = "123"

		Convey("no api authtoken", t, func(c C) {
			err := sender.Init(settings, logger, nil, "15:04")
			c.So(err, ShouldResemble, fmt.Errorf("can not read [%s] api_authtoken param from config", ""))
			c.So(sender, ShouldResemble, Sender{})
		})

		settings["api_authtoken"] = "321"

		Convey("no api fromphone", t, func(c C) {
			err := sender.Init(settings, logger, nil, "15:04")
			c.So(err, ShouldResemble, fmt.Errorf("can not read [%s] api_fromphone param from config", ""))
			c.So(sender, ShouldResemble, Sender{})
		})

		settings["api_fromphone"] = "12345678989"

		Convey("no api type", t, func(c C) {
			err := sender.Init(settings, logger, nil, "15:04")
			c.So(err, ShouldResemble, fmt.Errorf("wrong twilio type: %s", ""))
			c.So(sender, ShouldResemble, Sender{})
		})

		Convey("config sms", t, func(c C) {
			settings["type"] = "twilio sms"
			err := sender.Init(settings, logger, location, "15:04")
			c.So(err, ShouldBeNil)
			c.So(sender, ShouldResemble, Sender{sender: &twilioSenderSms{
				twilioSender{
					client:       twilio.NewClient("123", "321"),
					APIFromPhone: "12345678989",
					logger:       logger,
					location:     location,
				},
			}})
		})

		Convey("config voice", t, func(c C) {
			settings["type"] = "twilio voice"
			Convey("no voice url", t, func(c C) {
				err := sender.Init(settings, logger, location, "15:04")
				c.So(err, ShouldResemble, fmt.Errorf("can not read [%s] voiceurl param from config", "twilio voice"))
				c.So(sender, ShouldResemble, Sender{})
			})

			Convey("has voice url", t, func(c C) {
				settings["voiceurl"] = "url here"
				Convey("append_message == true", t, func(c C) {
					settings["append_message"] = "true"
					err := sender.Init(settings, logger, location, "15:04")
					c.So(err, ShouldBeNil)
					c.So(sender, ShouldResemble, Sender{sender: &twilioSenderVoice{
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

				Convey("append_message is something another string", t, func(c C) {
					settings["append_message"] = "something another string"
					err := sender.Init(settings, logger, location, "15:04")
					c.So(err, ShouldBeNil)
					c.So(sender, ShouldResemble, Sender{sender: &twilioSenderVoice{
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
