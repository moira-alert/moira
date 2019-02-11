package mail

import (
	"crypto/tls"
	"fmt"
	"html/template"
	"io"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"github.com/moira-alert/moira"
	"gopkg.in/gomail.v2"
)

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

// SendEvents implements Sender interface Send
func (sender *Sender) SendEvents(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plot []byte, throttled bool) error {
	message := sender.makeMessage(events, contact, trigger, plot, throttled)
	return sender.dialAndSend(message)
}

func (sender *Sender) makeMessage(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plot []byte, throttled bool) *gomail.Message {
	state := events.GetSubjectState()
	tags := trigger.GetTags()

	subject := fmt.Sprintf("%s %s %s (%d)", state, trigger.Name, tags, len(events))

	templateData := triggerData{
		Link:         trigger.GetTriggerURI(sender.FrontURI),
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
			Timestamp:  time.Unix(event.Timestamp, 0).In(sender.location).Format(sender.dateTimeFormat),
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

func formatDescription(desc string) template.HTML {
	escapedDesc := template.HTMLEscapeString(desc)
	escapedDesc = strings.Replace(escapedDesc, "\n", "\n<br/>", -1)

	return template.HTML(escapedDesc)
}

func (sender *Sender) dialAndSend(message *gomail.Message) error {
	d := gomail.Dialer{
		Host: sender.SMTPHost,
		Port: int(sender.SMTPPort),
		TLSConfig: &tls.Config{
			InsecureSkipVerify: sender.InsecureTLS,
			ServerName:         sender.SMTPHost,
		},
	}
	if sender.Password != "" {
		d.Auth = smtp.PlainAuth("", sender.Username, sender.Password, sender.SMTPHost)
	}
	if err := d.DialAndSend(message); err != nil {
		return err
	}
	return nil
}
