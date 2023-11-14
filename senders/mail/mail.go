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
	clients map[string]*mailClient
}

type mailClient struct {
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
	Template       *template.Template
	location       *time.Location
	dateTimeFormat string
	logger         moira.Logger
}

// Init read yaml config
func (sender *Sender) Init(opts moira.InitOptions) error {
	client := &mailClient{}
	senderIdent, err := client.fillSettings(opts.SenderSettings, opts.Logger, opts.Location, opts.DateTimeFormat)
	if err != nil {
		return err
	}

	client.TemplateName, client.Template, err = parseTemplate(client.TemplateFile)
	if err != nil {
		return err
	}

	err = client.tryDial()

	if sender.clients == nil {
		sender.clients = make(map[string]*mailClient)
	}

	sender.clients[senderIdent] = client

	return err
}

func (client *mailClient) fillSettings(senderSettings interface{}, logger moira.Logger, location *time.Location, dateTimeFormat string) (string, error) {
	var cfg config
	err := mapstructure.Decode(senderSettings, &cfg)
	if err != nil {
		return "", fmt.Errorf("failed to decode senderSettings to mail config: %w", err)
	}

	client.logger = logger
	client.From = cfg.MailFrom
	client.SMTPHello = cfg.SMTPHello
	client.SMTPHost = cfg.SMTPHost
	client.SMTPPort = cfg.SMTPPort
	client.InsecureTLS = cfg.InsecureTLS
	client.FrontURI = cfg.FrontURI
	client.Password = cfg.SMTPPass
	client.Username = cfg.SMTPUser
	client.TemplateFile = cfg.TemplateFile
	client.location = location
	client.dateTimeFormat = dateTimeFormat

	if client.Username == "" {
		client.Username = client.From
	}

	if client.From == "" {
		return "", fmt.Errorf("mail_from can't be empty")
	}

	var senderIdent string
	if cfg.Name != "" {
		senderIdent = cfg.Name
	} else {
		senderIdent = cfg.Type
	}

	return senderIdent, nil
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

func (client *mailClient) tryDial() error {
	t, err := smtp.Dial(fmt.Sprintf("%s:%d", client.SMTPHost, client.SMTPPort))
	if err != nil {
		return err
	}
	defer t.Close()

	if client.SMTPHello != "" {
		if err := t.Hello(client.SMTPHello); err != nil {
			return err
		}
	}

	if client.Password != "" {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: client.InsecureTLS,
			ServerName:         client.SMTPHost,
		}

		if err := t.StartTLS(tlsConfig); err != nil {
			return err
		}

		if err := t.Auth(smtp.PlainAuth("", client.Username, client.Password, client.SMTPHost)); err != nil {
			return err
		}
	}

	return nil
}
