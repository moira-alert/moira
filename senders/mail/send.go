package mail

import (
	"crypto/tls"
	"fmt"
	"html/template"
	"io"
	"net/smtp"
	"strconv"
	"strings"

	"github.com/russross/blackfriday/v2"

	"github.com/moira-alert/moira"
	"gopkg.in/gomail.v2"
)

type templateRow struct {
	Metric     string
	Timestamp  string
	Oldstate   moira.State
	State      moira.State
	Values     string
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
	TriggerState moira.State
	Items        []*templateRow
	PlotCID      string
}

// SendEvents implements Sender interface Send
func (sender *Sender) SendEvents(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plots [][]byte, throttled bool) error {
	client, ok := sender.clients[contact.Type]
	if !ok {
		return fmt.Errorf("failed to send events because there is not %s client", contact.Type)
	}

	message := client.makeMessage(events, contact, trigger, plots, throttled)
	return client.dialAndSend(message)
}

func (client *mailClient) makeMessage(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plots [][]byte, throttled bool) *gomail.Message {
	state := events.GetCurrentState(throttled)

	tags := trigger.GetTags()

	subject := fmt.Sprintf("%s %s %s (%d)", state, trigger.Name, tags, len(events))

	templateData := triggerData{
		Link:         trigger.GetTriggerURI(client.FrontURI),
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
			Timestamp:  event.FormatTimestamp(client.location, client.dateTimeFormat),
			Oldstate:   event.OldState,
			State:      event.State,
			Values:     event.GetMetricsValues(moira.DefaultNotificationSettings),
			WarnValue:  strconv.FormatFloat(trigger.WarnValue, 'f', -1, 64),
			ErrorValue: strconv.FormatFloat(trigger.ErrorValue, 'f', -1, 64),
			Message:    event.CreateMessage(client.location),
		})
	}

	m := gomail.NewMessage()
	m.SetHeader("From", client.From)
	m.SetHeader("To", contact.Value)
	m.SetHeader("Subject", subject)

	if len(plots) > 0 {
		for i, plot := range plots {
			plotCID := fmt.Sprintf("plot-t%d.png", i)
			templateData.PlotCID = plotCID
			m.Embed(plotCID, gomail.SetCopyFunc(func(w io.Writer) error {
				_, err := w.Write(plot)
				return err
			}))
		}
	}

	m.AddAlternativeWriter("text/html", func(w io.Writer) error {
		return client.Template.ExecuteTemplate(w, client.TemplateName, templateData)
	})

	return m
}

func formatDescription(desc string) template.HTML {
	htmlDesc := blackfriday.Run([]byte(desc))
	htmlDescWithbr := strings.Replace(string(htmlDesc), "\n", "<br/>", -1)
	return template.HTML(htmlDescWithbr)
}

func (client *mailClient) dialAndSend(message *gomail.Message) error {
	d := gomail.Dialer{
		Host:      client.SMTPHost,
		Port:      int(client.SMTPPort),
		LocalName: client.SMTPHello,
		TLSConfig: &tls.Config{
			InsecureSkipVerify: client.InsecureTLS,
			ServerName:         client.SMTPHost,
		},
	}

	if client.Password != "" {
		d.Auth = smtp.PlainAuth("", client.Username, client.Password, client.SMTPHost)
	}

	if err := d.DialAndSend(message); err != nil {
		return err
	}

	return nil
}
