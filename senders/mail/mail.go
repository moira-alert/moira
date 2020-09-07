package mail

import (
	"crypto/tls"
	"fmt"
	"html/template"
	"net/smtp"
	"path/filepath"
	"strconv"
	"time"

	"github.com/moira-alert/moira"
)

// Sender implements moira sender interface via pushover
type Sender struct {
	From           string
	SMTPHello      string
	SMTPHost       string
	SMTPPort       int64
	FrontURI       string
	InsecureTLS    bool
	Password       string
	Username       string
	TemplateFile   string
	TemplateName   string
	logger         moira.Logger
	Template       *template.Template
	location       *time.Location
	dateTimeFormat string
}

// Init read yaml config
func (sender *Sender) Init(senderSettings map[string]string, logger moira.Logger, location *time.Location, dateTimeFormat string) error {
	err := sender.fillSettings(senderSettings, logger, location, dateTimeFormat)
	if err != nil {
		return err
	}
	sender.TemplateName, sender.Template, err = parseTemplate(sender.TemplateFile)
	if err != nil {
		return err
	}
	err = sender.tryDial()
	return err
}

func (sender *Sender) fillSettings(senderSettings map[string]string, logger moira.Logger, location *time.Location, dateTimeFormat string) error {
	sender.logger = logger
	sender.From = senderSettings["mail_from"]
	sender.SMTPHello = senderSettings["smtp_hello"]
	sender.SMTPHost = senderSettings["smtp_host"]
	sender.SMTPPort, _ = strconv.ParseInt(senderSettings["smtp_port"], 10, 64)
	sender.InsecureTLS, _ = strconv.ParseBool(senderSettings["insecure_tls"])
	sender.FrontURI = senderSettings["front_uri"]
	sender.Password = senderSettings["smtp_pass"]
	sender.Username = senderSettings["smtp_user"]
	sender.TemplateFile = senderSettings["template_file"]
	sender.location = location
	sender.dateTimeFormat = dateTimeFormat
	if sender.Username == "" {
		sender.Username = sender.From
	}
	if sender.From == "" {
		return fmt.Errorf("mail_from can't be empty")
	}
	return nil
}

func parseTemplate(templateFilePath string) (name string, parsedTemplate *template.Template, err error) {
	if templateFilePath == "" {
		templateName := "mail" //nolint
		parsedTemplate, err = template.New(templateName).Parse(defaultTemplate)
		return templateName, parsedTemplate, err
	}
	templateName := filepath.Base(templateFilePath)
	parsedTemplate, err = template.New(templateName).Funcs(template.FuncMap{
		"htmlSafe": func(html string) template.HTML {
			return template.HTML(html)
		},
	}).ParseFiles(templateFilePath)
	return templateName, parsedTemplate, err
}

func (sender *Sender) tryDial() error {
	t, err := smtp.Dial(fmt.Sprintf("%s:%d", sender.SMTPHost, sender.SMTPPort))
	if err != nil {
		return err
	}
	defer t.Close()
	if sender.SMTPHello != "" {
		if err := t.Hello(sender.SMTPHello); err != nil {
			return err
		}
	}
	if sender.Password != "" {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: sender.InsecureTLS,
			ServerName:         sender.SMTPHost,
		}
		if err := t.StartTLS(tlsConfig); err != nil {
			return err
		}
		if err := t.Auth(smtp.PlainAuth("", sender.Username, sender.Password, sender.SMTPHost)); err != nil {
			return err
		}
	}
	return nil
}
