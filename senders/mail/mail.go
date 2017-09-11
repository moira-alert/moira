package mail

import (
	"crypto/tls"
	"fmt"
	"html/template"
	"io"
	"net/smtp"
	"strconv"
	"time"

	"github.com/moira-alert/moira-alert"
	gomail "gopkg.in/gomail.v2"
)

// Sender implements moira sender interface via pushover
type Sender struct {
	From         string
	SMTPhost     string
	SMTPport     int64
	FrontURI     string
	InsecureTLS  bool
	Password     string
	Username     string
	TemplateFile string
	log          moira.Logger
	Template     *template.Template
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

// Init read yaml config
func (sender *Sender) Init(senderSettings map[string]string, logger moira.Logger) error {
	sender.setLogger(logger)
	sender.From = senderSettings["mail_from"]
	sender.SMTPhost = senderSettings["smtp_host"]
	sender.SMTPport, _ = strconv.ParseInt(senderSettings["smtp_port"], 10, 64)
	sender.InsecureTLS, _ = strconv.ParseBool(senderSettings["insecure_tls"])
	sender.FrontURI = senderSettings["front_uri"]
	sender.Password = senderSettings["smtp_pass"]
	sender.Username = senderSettings["smtp_user"]
	sender.TemplateFile = senderSettings["template_file"]

	if sender.Username == "" {
		sender.Username = sender.From
	}
	if sender.From == "" {
		return fmt.Errorf("mail_from can't be empty")
	}

	if sender.TemplateFile == "" {
		sender.Template = template.Must(template.New("mail").Parse(defaultTemplate))
	} else {
		var err error
		if sender.Template, err = template.New("mail").ParseFiles(sender.TemplateFile); err != nil {
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
func (sender *Sender) SendEvents(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, throttled bool) error {

	m := sender.makeMessage(events, contact, trigger, throttled)

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

func (sender *Sender) makeMessage(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, throttled bool) *gomail.Message {
	state := events.GetSubjectState()
	tags := trigger.GetTags()

	subject := fmt.Sprintf("%s %s %s (%d)", state, trigger.Name, tags, len(events))

	templateData := struct {
		Link        string
		Description string
		Throttled   bool
		Items       []*templateRow
	}{
		Link:        fmt.Sprintf("%s/#/events/%s", sender.FrontURI, events[0].TriggerID),
		Description: trigger.Desc,
		Throttled:   throttled,
		Items:       make([]*templateRow, 0, len(events)),
	}

	for _, event := range events {
		templateData.Items = append(templateData.Items, &templateRow{
			Metric:     event.Metric,
			Timestamp:  time.Unix(event.Timestamp, 0).Format("15:04 02.01.2006"),
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
	m.AddAlternativeWriter("text/html", func(w io.Writer) error {
		return sender.Template.Execute(w, templateData)
	})

	return m
}

func (sender *Sender) setLogger(logger moira.Logger) {
	sender.log = logger
}
