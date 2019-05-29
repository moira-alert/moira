package pagerduty

import (
	"bytes"
	"fmt"

	"github.com/PagerDuty/go-pagerduty"

	"github.com/moira-alert/moira"
)

const summaryMaxChars = 1024

// SendEvents implements Sender interface Send
func (sender *Sender) SendEvents(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plot []byte, throttled bool) error {
	event := sender.buildEvent(events, contact, trigger, plot, throttled)
	_, err := pagerduty.ManageEvent(event)
	if err != nil {
		return fmt.Errorf("failed to post the event to the pagerduty contact %s : %s. ", contact.Value, err)
	}
	return nil
}

func (sender *Sender) buildSummary(events moira.NotificationEvents, trigger moira.TriggerData, throttled bool) string {
	var summary bytes.Buffer

	summary.WriteString(fmt.Sprintf("*%s*", events.GetSubjectState()))

	tags := trigger.GetTags()
	if tags != "" {
		summary.WriteString(" ")
		summary.WriteString(tags)
	}

	triggerURI := trigger.GetTriggerURI(sender.frontURI)
	if triggerURI != "" {
		summary.WriteString(triggerURI)
	} else if trigger.Name != "" {
		summary.WriteString(" ")
		summary.WriteString(trigger.Name)
	}

	if trigger.Desc != "" {
		summary.WriteString("\n")
		// TODO: string MD before writing
		summary.WriteString(trigger.Desc)
	}

	summary.WriteString("\n")

	var printEventsCount int
	summaryCharCount := len([]rune(summary.String()))
	messageLimitReached := false

	for _, event := range events {
		line := fmt.Sprintf("\n%s: %s = %s (%s to %s)", event.FormatTimestamp(sender.location), event.Metric, event.GetMetricValue(), event.OldState, event.State)
		if len(moira.UseString(event.Message)) > 0 {
			line += fmt.Sprintf(". %s", moira.UseString(event.Message))
		}
		lineCharsCount := len([]rune(line))
		if summaryCharCount+lineCharsCount > summaryMaxChars {
			messageLimitReached = true
			break
		}
		summary.WriteString(line)
		summaryCharCount += lineCharsCount
		printEventsCount++
	}

	if messageLimitReached {
		summary.WriteString(fmt.Sprintf("\n\n...and %d more events.", len(events)-printEventsCount))
	}

	if throttled {
		summary.WriteString("\nPlease, *fix your system or tune this trigger* to generate less events.")
	}
	return summary.String()
}
func (sender *Sender) buildEvent(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plot []byte, throttled bool) pagerduty.V2Event {
	if len(plot) > 0 {
		// attach image to the event
	}
	summary := sender.buildSummary(events, trigger, throttled)
	payload := &pagerduty.V2Payload{
		Summary:  summary,
		Severity: "info",
		Source:   "moira",
	}
	event := pagerduty.V2Event{
		RoutingKey: contact.Value,
		Action:     "trigger",
		Payload:    payload,
	}
	return event
}

// func (sender *Sender) sendPlot(plot []byte, channelID, threadTimestamp, triggerID string) error {
// 	reader := bytes.NewReader(plot)
// 	uploadParameters := slack.FileUploadParameters{
// 		Channels:        []string{channelID},
// 		ThreadTimestamp: threadTimestamp,
// 		Reader:          reader,
// 		Filetype:        "png",
// 		Filename:        fmt.Sprintf("%s.png", triggerID),
// 	}
// 	_, err := sender.client.UploadFile(uploadParameters)
// 	return err
// }
