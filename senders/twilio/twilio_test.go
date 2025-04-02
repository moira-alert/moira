package twilio

import (
	"errors"
	"fmt"
	"testing"
	"time"

	twilio_client "github.com/carlosdp/twiliogo"
	"github.com/go-playground/validator/v10"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	. "github.com/smartystreets/goconvey/convey"
)

func TestInit(t *testing.T) {
	Convey("Tests init twilio sender", t, func() {
		sender := Sender{}
		logger, _ := logging.ConfigureLog("stdout", "debug", "test", true)
		location, _ := time.LoadLocation("UTC")
		settings := map[string]interface{}{}

		validatorErr := validator.ValidationErrors{}

		Convey("no api asid", func() {
			err := sender.Init(settings, logger, nil, "15:04")
			So(errors.As(err, &validatorErr), ShouldBeTrue)
			So(sender, ShouldResemble, Sender{})
		})

		settings["api_asid"] = "123"

		Convey("no api authtoken", func() {
			err := sender.Init(settings, logger, nil, "15:04")
			So(errors.As(err, &validatorErr), ShouldBeTrue)
			So(sender, ShouldResemble, Sender{})
		})

		settings["api_authtoken"] = "321"

		Convey("no api fromphone", func() {
			err := sender.Init(settings, logger, nil, "15:04")
			So(errors.As(err, &validatorErr), ShouldBeTrue)
			So(sender, ShouldResemble, Sender{})
		})

		settings["api_fromphone"] = "12345678989"

		Convey("no api type", func() {
			err := sender.Init(settings, logger, nil, "15:04")
			So(errors.As(err, &validatorErr), ShouldBeTrue)
			So(sender, ShouldResemble, Sender{})
		})

		settings["sender_type"] = "test"

		Convey("with unknown api type", func() {
			err := sender.Init(settings, logger, nil, "15:04")
			So(err, ShouldResemble, fmt.Errorf("wrong twilio type: %s", "test"))
			So(sender, ShouldResemble, Sender{})
		})

		Convey("config sms", func() {
			settings["sender_type"] = "twilio sms"
			err := sender.Init(settings, logger, location, "15:04")
			So(err, ShouldBeNil)
			So(sender, ShouldResemble, Sender{sender: &twilioSenderSms{
				twilioSender{
					client:       twilio_client.NewClient("123", "321"),
					APIFromPhone: "12345678989",
					logger:       logger,
					location:     location,
				},
			}})
		})

		Convey("config voice", func() {
			settings["sender_type"] = "twilio voice"
			Convey("no voice url", func() {
				err := sender.Init(settings, logger, location, "15:04")
				So(err, ShouldResemble, fmt.Errorf("can not read [%s] voiceurl param from config", "twilio voice"))
				So(sender, ShouldResemble, Sender{})
			})

			Convey("has voice url", func() {
				settings["voiceurl"] = "url here"
				Convey("append_message == true", func() {
					settings["append_message"] = true
					err := sender.Init(settings, logger, location, "15:04")
					So(err, ShouldBeNil)
					So(sender, ShouldResemble, Sender{sender: &twilioSenderVoice{
						twilioSender: twilioSender{
							client:       twilio_client.NewClient("123", "321"),
							APIFromPhone: "12345678989",
							logger:       logger,
							location:     location,
						},
						voiceURL:      "url here",
						appendMessage: true,
					}})
				})

				Convey("append_message is false", func() {
					settings["append_message"] = false
					err := sender.Init(settings, logger, location, "15:04")
					So(err, ShouldBeNil)
					So(sender, ShouldResemble, Sender{sender: &twilioSenderVoice{
						twilioSender: twilioSender{
							client:       twilio_client.NewClient("123", "321"),
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
