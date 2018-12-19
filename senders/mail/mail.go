package mail

import (
	"crypto/tls"
	"fmt"
	"html/template"
	"io"
	"net/smtp"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/moira-alert/moira"
	"gopkg.in/gomail.v2"
)

// Sender implements moira sender interface via pushover
type Sender struct {
	From           string
	SMTPhost       string
	SMTPport       int64
	FrontURI       string
	InsecureTLS    bool
	Password       string
	Username       string
	TemplateFile   string
	TemplateName   string
	log            moira.Logger
	Template       *template.Template
	location       *time.Location
	DateTimeFormat string
}

type templateRow struct {
	Metric     string
	Timestamp  string
	Oldstate   string
	State      string
	Value      string
	WarnValue  string
	ErrorValue string
	Message    string
}

type triggerData struct {
	Link         string
	Description  template.HTML
	Throttled    bool
	TriggerName  string
	Tags         string
	TriggerState string
	Items        []*templateRow
	PlotCID      string
}

// Init read yaml config
func (sender *Sender) Init(senderSettings map[string]string, logger moira.Logger, location *time.Location, dateTimeFormat string) error {

	sender.setLogger(logger)
	sender.From = senderSettings["mail_from"]
	sender.SMTPhost = senderSettings["smtp_host"]
	sender.SMTPport, _ = strconv.ParseInt(senderSettings["smtp_port"], 10, 64)
	sender.InsecureTLS, _ = strconv.ParseBool(senderSettings["insecure_tls"])
	sender.FrontURI = senderSettings["front_uri"]
	sender.Password = senderSettings["smtp_pass"]
	sender.Username = senderSettings["smtp_user"]
	sender.TemplateFile = senderSettings["template_file"]
	sender.location = location
	sender.DateTimeFormat = dateTimeFormat

	if sender.Username == "" {
		sender.Username = sender.From
	}
	if sender.From == "" {
		return fmt.Errorf("mail_from can't be empty")
	}

	if sender.TemplateFile == "" {
		sender.TemplateName = "mail"
		sender.Template = template.Must(template.New(sender.TemplateName).Parse(defaultTemplate))
	} else {
		var err error

		sender.TemplateName = filepath.Base(sender.TemplateFile)
		sender.Template, err = template.New(sender.TemplateName).Funcs(template.FuncMap{
			"htmlSafe": func(html string) template.HTML {
				return template.HTML(html)
			},
		}).ParseFiles(sender.TemplateFile)

		if err != nil {
			return err
		}
	}

	t, err := smtp.Dial(fmt.Sprintf("%s:%d", sender.SMTPhost, sender.SMTPport))
	if err != nil {
		return err
	}
	defer t.Close()
	if sender.Password != "" {
		if err := t.StartTLS(&tls.Config{
			InsecureSkipVerify: sender.InsecureTLS,
			ServerName:         sender.SMTPhost,
		}); err != nil {
			return err
		}
		if err := t.Auth(smtp.PlainAuth("", sender.Username, sender.Password, sender.SMTPhost)); err != nil {
			return err
		}
	}
	return nil
}

// SendEvents implements Sender interface Send
func (sender *Sender) SendEvents(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plot []byte, throttled bool) error {

	m := sender.makeMessage(events, contact, trigger, plot, throttled)

	d := gomail.Dialer{
		Host: sender.SMTPhost,
		Port: int(sender.SMTPport),
		TLSConfig: &tls.Config{
			InsecureSkipVerify: sender.InsecureTLS,
			ServerName:         sender.SMTPhost,
		},
	}

	if sender.Password != "" {
		d.Auth = smtp.PlainAuth("", sender.Username, sender.Password, sender.SMTPhost)
	}

	if err := d.DialAndSend(m); err != nil {
		return err
	}
	return nil
}

func (sender *Sender) makeMessage(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plot []byte, throttled bool) *gomail.Message {

	state := events.GetSubjectState()
	tags := trigger.GetTags()

	subject := fmt.Sprintf("%s %s %s (%d)", state, trigger.Name, tags, len(events))

	templateData := triggerData{
		Link:         fmt.Sprintf("%s/trigger/%s", sender.FrontURI, events[0].TriggerID),
		Description:  formatDescription(trigger.Desc),
		Throttled:    throttled,
		TriggerName:  trigger.Name,
		Tags:         tags,
		TriggerState: state,
		Items:        make([]*templateRow, 0, len(events)),
	}

	for _, event := range events {
		templateData.Items = append(templateData.Items, &templateRow{
			Metric:     event.Metric,
			Timestamp:  time.Unix(event.Timestamp, 0).In(sender.location).Format(sender.DateTimeFormat),
			Oldstate:   event.OldState,
			State:      event.State,
			Value:      strconv.FormatFloat(moira.UseFloat64(event.Value), 'f', -1, 64),
			WarnValue:  strconv.FormatFloat(trigger.WarnValue, 'f', -1, 64),
			ErrorValue: strconv.FormatFloat(trigger.ErrorValue, 'f', -1, 64),
			Message:    moira.UseString(event.Message),
		})
	}

	m := gomail.NewMessage()
	m.SetHeader("From", sender.From)
	m.SetHeader("To", contact.Value)
	m.SetHeader("Subject", subject)

	if len(plot) > 0 {
		plotCID := "plot.png"
		templateData.PlotCID = plotCID
		m.Embed(plotCID, gomail.SetCopyFunc(func(w io.Writer) error {
			_, err := w.Write(plot)
			return err
		}))
	}

	m.AddAlternativeWriter("text/html", func(w io.Writer) error {
		return sender.Template.ExecuteTemplate(w, sender.TemplateName, templateData)
	})

	return m
}

func (sender *Sender) setLogger(logger moira.Logger) {
	sender.log = logger
}

func formatDescription(desc string) template.HTML {
	escapedDesc := template.HTMLEscapeString(desc)
	escapedDesc = strings.Replace(escapedDesc, "\n", "\n<br/>", -1)

	return template.HTML(escapedDesc)
}
