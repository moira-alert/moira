package mail

import (
	"errors"
	"testing"

	"github.com/go-playground/validator/v10"
	. "github.com/smartystreets/goconvey/convey"
)

const (
	defaultMailFrom     = "test-mail-from"
	defaultSMTPHost     = "test-smtp-host"
	defaultSMTPPort     = 80
	defaultSMTPHello    = "test-smtp-hello"
	defaultInsecureTLS  = true
	defaultFrontURI     = "test-front-uri"
	defaultSMTPPass     = "test-smtp-pass"
	defaultSMTPUser     = "test-smtp-user"
	defaultTemplateFile = "test-template-file"
)

func TestFillSettings(t *testing.T) {
	Convey("Test fillSettings", t, func() {
		sender := Sender{}

		validatorErr := validator.ValidationErrors{}

		Convey("With empty mail_from", func() {
			senderSettings := map[string]interface{}{
				"smtp_host": defaultSMTPHost,
				"smtp_port": defaultSMTPPort,
			}

			err := sender.fillSettings(senderSettings, nil, nil, "")
			So(errors.As(err, &validatorErr), ShouldBeTrue)
			So(sender, ShouldResemble, Sender{})
		})

		Convey("With empty smpt_host", func() {
			senderSettings := map[string]interface{}{
				"mail_from": defaultMailFrom,
				"smtp_port": defaultSMTPPort,
			}

			err := sender.fillSettings(senderSettings, nil, nil, "")
			So(errors.As(err, &validatorErr), ShouldBeTrue)
			So(sender, ShouldResemble, Sender{})
		})

		Convey("With empty smpt_port", func() {
			senderSettings := map[string]interface{}{
				"mail_from": defaultMailFrom,
				"smtp_host": defaultSMTPHost,
			}

			err := sender.fillSettings(senderSettings, nil, nil, "")
			So(errors.As(err, &validatorErr), ShouldBeTrue)
			So(sender, ShouldResemble, Sender{})
		})

		Convey("With full settings", func() {
			senderSettings := map[string]interface{}{
				"mail_from":     defaultMailFrom,
				"smtp_host":     defaultSMTPHost,
				"smtp_port":     defaultSMTPPort,
				"smtp_hello":    defaultSMTPHello,
				"insecure_tls":  defaultInsecureTLS,
				"front_uri":     defaultFrontURI,
				"smtp_user":     defaultSMTPUser,
				"smtp_pass":     defaultSMTPPass,
				"template_file": defaultTemplateFile,
			}

			err := sender.fillSettings(senderSettings, nil, nil, "")
			So(err, ShouldBeNil)
			So(sender, ShouldResemble, Sender{
				From:         defaultMailFrom,
				SMTPHello:    defaultSMTPHello,
				SMTPHost:     defaultSMTPHost,
				SMTPPort:     80,
				FrontURI:     defaultFrontURI,
				InsecureTLS:  defaultInsecureTLS,
				Username:     defaultSMTPUser,
				Password:     defaultSMTPPass,
				TemplateFile: defaultTemplateFile,
			})
		})
	})
}

func TestParseTemplate(t *testing.T) {
	Convey("Template path is empty", t, func() {
		name, t, err := parseTemplate("")
		So(name, ShouldResemble, "mail")
		So(t, ShouldNotBeNil)
		So(err, ShouldBeNil)
	})

	Convey("Template path no empty", t, func() {
		name, t, err := parseTemplate("bin/template")
		So(name, ShouldResemble, "template")
		So(t, ShouldBeNil)
		So(err, ShouldNotBeNil)
	})
}
