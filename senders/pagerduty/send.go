package pagerduty

import (
	"bytes"
	"context"
	"fmt"
	"time"

	stripmd "github.com/writeas/go-strip-markdown"

	"github.com/PagerDuty/go-pagerduty"

	"github.com/moira-alert/moira"
)

const summaryMaxChars = 1024

// SendEvents implements Sender interface Send.
func (sender *Sender) SendEvents(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plots [][]byte, throttled bool) error {
	event := sender.buildEvent(events, contact, trigger, plots, throttled)

	_, err := pagerduty.ManageEventWithContext(context.Background(), event)
	if err != nil {
		return fmt.Errorf("failed to post the event to the pagerduty contact %s : %w. ", contact.Value, err)
	}

	return nil
}

func (sender *Sender) buildEvent(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plots [][]byte, throttled bool) pagerduty.V2Event {
	summary := sender.buildSummary(events, trigger, throttled)
	details := make(map[string]interface{})

	details["Trigger Name"] = trigger.Name
	triggerURI := trigger.GetTriggerURI(sender.frontURI)

	if triggerURI != "" {
		details["Trigger URI"] = triggerURI
	}

	desc := trigger.Desc
	if contact.ExtraMessage != "" {
		desc = contact.ExtraMessage + "\n" + desc
	}

	if desc != "" {
		details["Description"] = stripmd.Strip(desc)
	}

	var eventList string

	for _, event := range events {
		line := fmt.Sprintf("\n%s: %s = %s (%s to %s)", event.FormatTimestamp(sender.location, moira.DefaultTimeFormat), event.Metric, event.GetMetricsValues(moira.DefaultNotificationSettings), event.OldState, event.State)
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

	if len(plots) > 0 && sender.imageStoreConfigured {
		for i, plot := range plots {
			imageLink, err := sender.imageStore.StoreImage(plot)
			if err != nil {
				sender.logger.Warning().
					Error(err).
					Msg("could not store the plot image in the image store")
			} else {
				imageDetails := map[string]string{
					"src": imageLink,
					"alt": fmt.Sprintf("Plot-%d", i),
				}
				event.Images = append(event.Images, imageDetails)
			}
		}
	}

	return event
}

func (sender *Sender) getSeverity(events moira.NotificationEvents) string {
	severity := "info"

	for _, event := range events {
		if event.State == moira.StateERROR || event.State == moira.StateEXCEPTION {
			severity = "error"
		}

		if severity != "error" && (event.State == moira.StateWARN || event.State == moira.StateNODATA) {
			severity = "warning"
		}
	}

	return severity
}

func (sender *Sender) buildSummary(events moira.NotificationEvents, trigger moira.TriggerData, throttled bool) string {
	var summary bytes.Buffer

	state := events.GetCurrentState(throttled)

	summary.WriteString(string(state))
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
