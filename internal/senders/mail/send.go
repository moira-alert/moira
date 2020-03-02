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

	moira2 "github.com/moira-alert/moira/internal/moira"

	"github.com/russross/blackfriday/v2"

	"gopkg.in/gomail.v2"
)

type templateRow struct {
	Metric     string
	Timestamp  string
	Oldstate   moira2.State
	State      moira2.State
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
	TriggerState moira2.State
	Items        []*templateRow
	PlotCID      string
}

// SendEvents implements Sender interface Send
func (sender *Sender) SendEvents(events moira2.NotificationEvents, contact moira2.ContactData, trigger moira2.TriggerData, plot []byte, throttled bool) error {
	message := sender.makeMessage(events, contact, trigger, plot, throttled)
	return sender.dialAndSend(message)
}

func (sender *Sender) makeMessage(events moira2.NotificationEvents, contact moira2.ContactData, trigger moira2.TriggerData, plot []byte, throttled bool) *gomail.Message {
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
			Value:      strconv.FormatFloat(moira2.UseFloat64(event.Value), 'f', -1, 64),
			WarnValue:  strconv.FormatFloat(trigger.WarnValue, 'f', -1, 64),
			ErrorValue: strconv.FormatFloat(trigger.ErrorValue, 'f', -1, 64),
			Message:    event.CreateMessage(sender.location),
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
	htmlDesc := blackfriday.Run([]byte(desc))
	htmlDescWithbr := strings.Replace(string(htmlDesc), "\n", "<br/>", -1)
	return template.HTML(htmlDescWithbr)
}

func (sender *Sender) dialAndSend(message *gomail.Message) error {
	d := gomail.Dialer{
		Host:      sender.SMTPHost,
		Port:      int(sender.SMTPPort),
		LocalName: sender.SMTPHello,
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
