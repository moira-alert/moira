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

func (sender *Sender) buildEvent(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plot []byte, throttled bool) pagerduty.V2Event {
	summary := sender.buildSummary(events, trigger, throttled)
	details := make(map[string]interface{})

	triggerURI := trigger.GetTriggerURI(sender.frontURI)
	if triggerURI != "" {
		details["Trigger URI"] = triggerURI
	} else if trigger.Name != "" {
		details["Trigger URI"] = trigger.Name
	}

	if trigger.Desc != "" {
		// TODO: string MD before writing
		details["Description"] = trigger.Desc
	}

	var eventList string

	for _, event := range events {
		line := fmt.Sprintf("\n%s: %s = %s (%s to %s)", event.FormatTimestamp(sender.location), event.Metric, event.GetMetricValue(), event.OldState, event.State)
		if len(moira.UseString(event.Message)) > 0 {
			line += fmt.Sprintf(". %s", moira.UseString(event.Message))
		}
		eventList += line
	}

	details["Events"] = eventList
	if throttled {
		details["Throttled"] = "Please, fix your system or tune this trigger to generate less events."
	}
	payload := &pagerduty.V2Payload{
		Summary:  summary,
		Severity: "info",
		Source:   "moira",
		Details:  details,
	}
	event := pagerduty.V2Event{
		RoutingKey: contact.Value,
		Action:     "trigger",
		Payload:    payload,
	}

	if len(plot) > 0 {
		imageLink, err := sender.ImageStore.StoreImage(plot)
		if err != nil {
			sender.logger.Warningf("could not store the plot image in the image store: %s", err)
		} else {
			event.Images = append(event.Images, imageLink)
		}
	}

	return event
}

func (sender *Sender) buildSummary(events moira.NotificationEvents, trigger moira.TriggerData, throttled bool) string {
	var summary bytes.Buffer

	summary.WriteString(fmt.Sprintf("%s", events.GetSubjectState()))

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
