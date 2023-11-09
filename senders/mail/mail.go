package mail

import (
	"crypto/tls"
	"fmt"
	"html/template"
	"net/smtp"
	"path/filepath"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/moira-alert/moira"
)

// Structure that represents the Mail configuration in the YAML file
type config struct {
	Name         string `mapstructure:"name"`
	Type         string `mapstructure:"type"`
	MailFrom     string `mapstructure:"mail_from"`
	SMTPHello    string `mapstructure:"smtp_hello"`
	SMTPHost     string `mapstructure:"smtp_host"`
	SMTPPort     int64  `mapstructure:"smtp_port"`
	InsecureTLS  bool   `mapstructure:"insecure_tls"`
	FrontURI     string `mapstructure:"front_uri"`
	SMTPPass     string `mapstructure:"smtp_pass"`
	SMTPUser     string `mapstructure:"smtp_user"`
	TemplateFile string `mapstructure:"template_file"`
}

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
func (sender *Sender) Init(senderSettings interface{}, logger moira.Logger, location *time.Location, dateTimeFormat string, sendersNameToType map[string]string) error {
	err := sender.fillSettings(senderSettings, logger, location, dateTimeFormat, sendersNameToType)
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

func (sender *Sender) fillSettings(senderSettings interface{}, logger moira.Logger, location *time.Location, dateTimeFormat string, sendersNameToType map[string]string) error {
	var cfg config
	err := mapstructure.Decode(senderSettings, &cfg)
	if err != nil {
		return fmt.Errorf("failed to decode senderSettings to mail config: %w", err)
	}

	if cfg.Name != "" {
		sendersNameToType[cfg.Name] = cfg.Type
	} else {
		sendersNameToType[cfg.Type] = cfg.Type
	}

	sender.logger = logger
	sender.From = cfg.MailFrom
	sender.SMTPHello = cfg.SMTPHello
	sender.SMTPHost = cfg.SMTPHost
	sender.SMTPPort = cfg.SMTPPort
	sender.InsecureTLS = cfg.InsecureTLS
	sender.FrontURI = cfg.FrontURI
	sender.Password = cfg.SMTPPass
	sender.Username = cfg.SMTPUser
	sender.TemplateFile = cfg.TemplateFile
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
