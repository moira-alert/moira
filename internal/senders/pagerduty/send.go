package pagerduty

import (
	"bytes"
	"fmt"
	"time"

	moira2 "github.com/moira-alert/moira/internal/moira"

	stripmd "github.com/writeas/go-strip-markdown"

	"github.com/PagerDuty/go-pagerduty"
)

const summaryMaxChars = 1024

// SendEvents implements Sender interface Send
func (sender *Sender) SendEvents(events moira2.NotificationEvents, contact moira2.ContactData, trigger moira2.TriggerData, plot []byte, throttled bool) error {
	event := sender.buildEvent(events, contact, trigger, plot, throttled)
	_, err := pagerduty.ManageEvent(event)
	if err != nil {
		return fmt.Errorf("failed to post the event to the pagerduty contact %s : %s. ", contact.Value, err)
	}
	return nil
}

func (sender *Sender) buildEvent(events moira2.NotificationEvents, contact moira2.ContactData, trigger moira2.TriggerData, plot []byte, throttled bool) pagerduty.V2Event {
	summary := sender.buildSummary(events, trigger, throttled)
	details := make(map[string]interface{})

	details["Trigger Name"] = trigger.Name
	triggerURI := trigger.GetTriggerURI(sender.frontURI)
	if triggerURI != "" {
		details["Trigger URI"] = triggerURI
	}

	if trigger.Desc != "" {
		details["Description"] = stripmd.Strip(trigger.Desc)
	}

	var eventList string

	for _, event := range events {
		line := fmt.Sprintf("\n%s: %s = %s (%s to %s)", event.FormatTimestamp(sender.location), event.Metric, event.GetMetricValue(), event.OldState, event.State)
		if msg := event.CreateMessage(sender.location); len(msg) > 0 {
			line += fmt.Sprintf(". %s", msg)
		}
		eventList += line
	}

	details["Events"] = eventList
	if throttled {
		details["Message"] = "Please, fix your system or tune this trigger to generate less events."
	}

	payload := &pagerduty.V2Payload{
		Summary:   summary,
		Severity:  sender.getSeverity(events),
		Source:    "moira",
		Timestamp: time.Unix(events[len(events)-1].Timestamp, 0).UTC().Format(time.RFC3339),
		Details:   details,
	}

	event := pagerduty.V2Event{
		RoutingKey: contact.Value,
		Action:     "trigger",
		Payload:    payload,
	}

	if len(plot) > 0 && sender.imageStoreConfigured {
		imageLink, err := sender.imageStore.StoreImage(plot)
		if err != nil {
			sender.logger.Warningf("could not store the plot image in the image store: %s", err)
		} else {
			imageDetails := map[string]string{
				"src": imageLink,
				"alt": "Plot",
			}
			event.Images = append(event.Images, imageDetails)
		}
	}

	return event
}

func (sender *Sender) getSeverity(events moira2.NotificationEvents) string {
	severity := "info"
	for _, event := range events {
		if event.State == moira2.StateERROR || event.State == moira2.StateEXCEPTION {
			severity = "error"
		}
		if severity != "error" && (event.State == moira2.StateWARN || event.State == moira2.StateNODATA) {
			severity = "warning"
		}
	}
	return severity
}

func (sender *Sender) buildSummary(events moira2.NotificationEvents, trigger moira2.TriggerData, throttled bool) string {
	var summary bytes.Buffer

	summary.WriteString(string(events.GetSubjectState()))
	summary.WriteString(" ")
	summary.WriteString(trigger.Name)

	tags := trigger.GetTags()
	if tags != "" {
		summary.WriteString(" ")
		summary.WriteString(tags)
	}
	if len(summary.String()) > summaryMaxChars {
		summaryStr := summary.String()[:1000]
		summaryStr += "..."
		return summaryStr
	}
	return summary.String()
}
