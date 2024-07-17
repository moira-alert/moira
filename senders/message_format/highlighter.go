package message_format

import (
	"fmt"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/senders"
	"github.com/moira-alert/moira/senders/emoji_provider"
	"strings"
	"time"
)

const quotas = "```"

// HighlightSyntaxFormatter formats message by using functions, emojis and some other highlight patterns.
type HighlightSyntaxFormatter struct {
	// EmojiGetter used in titles for better description.
	EmojiGetter emoji_provider.StateEmojiGetter
	FrontURI    string
	Location    *time.Location
	UseEmoji    bool
	// UriFormatter is used for formatting uris, for example for Markdown use something like
	// fmt.Sprintf("[%s](%s)", triggerName, triggerURI).
	UriFormatter func(triggerURI, triggerName string) string
	// DescriptionFormatter is used to format trigger description to supported description.
	DescriptionFormatter func(trigger moira.TriggerData) string
	// BoldFormatter makes str bold. For example in Markdown it should be **str**.
	BoldFormatter func(str string) string
}

func (formatter HighlightSyntaxFormatter) Format(params MessageFormatterParams) string {
	var message strings.Builder
	state := params.Events.GetCurrentState(params.Throttled)
	emoji := formatter.EmojiGetter.GetStateEmoji(state)

	title := formatter.buildTitle(params.Events, params.Trigger, emoji, params.Throttled)
	titleLen := len([]rune(title))

	desc := formatter.DescriptionFormatter(params.Trigger)
	descLen := len([]rune(desc))

	eventsString := formatter.buildEventsString(params.Events, -1, params.Throttled)
	eventsStringLen := len([]rune(eventsString))

	charsLeftAfterTitle := params.MessageMaxChars - titleLen

	descNewLen, eventsNewLen := senders.CalculateMessagePartsLength(charsLeftAfterTitle, descLen, eventsStringLen)
	if descLen != descNewLen {
		desc = desc[:descNewLen] + "...\n"
	}
	if eventsNewLen != eventsStringLen {
		eventsString = formatter.buildEventsString(params.Events, eventsNewLen, params.Throttled)
	}

	message.WriteString(title)
	message.WriteString(desc)
	message.WriteString(eventsString)
	return message.String()
}

func (formatter HighlightSyntaxFormatter) buildTitle(events moira.NotificationEvents, trigger moira.TriggerData, emoji string, throttled bool) string {
	state := events.GetCurrentState(throttled)
	title := ""
	if formatter.UseEmoji {
		title += emoji + " "
	}

	title += formatter.BoldFormatter(string(state))
	triggerURI := trigger.GetTriggerURI(formatter.FrontURI)
	if triggerURI != "" {
		title += fmt.Sprintf(" %s", formatter.UriFormatter(triggerURI, trigger.Name))
	} else if trigger.Name != "" {
		title += " " + trigger.Name
	}

	tags := trigger.GetTags()
	if tags != "" {
		title += " " + tags
	}

	title += "\n"
	return title
}

// buildEventsString builds the string from moira events and limits it to charsForEvents.
// if charsForEvents is negative buildEventsString does not limit the events string.
func (formatter HighlightSyntaxFormatter) buildEventsString(events moira.NotificationEvents, charsForEvents int, throttled bool) string {
	charsForThrottleMsg := 0
	throttleMsg := "\nPlease, *fix your system or tune this trigger* to generate less events."
	if throttled {
		charsForThrottleMsg = len([]rune(throttleMsg))
	}
	charsLeftForEvents := charsForEvents - charsForThrottleMsg

	var eventsString string
	eventsString += quotas
	var tailString string

	eventsLenLimitReached := false
	eventsPrinted := 0
	for _, event := range events {
		line := fmt.Sprintf(
			"\n%s: %s = %s (%s to %s)",
			event.FormatTimestamp(formatter.Location, moira.DefaultTimeFormat),
			event.Metric,
			event.GetMetricsValues(moira.DefaultNotificationSettings),
			event.OldState,
			event.State)
		if msg := event.CreateMessage(formatter.Location); len(msg) > 0 {
			line += fmt.Sprintf(". %s", msg)
		}

		tailString = fmt.Sprintf("\n...and %d more events.", len(events)-eventsPrinted)
		tailStringLen := len([]rune(quotas)) + len([]rune(tailString))
		if !(charsForEvents < 0) && (len([]rune(eventsString))+len([]rune(line)) > charsLeftForEvents-tailStringLen) {
			eventsLenLimitReached = true
			break
		}

		eventsString += line
		eventsPrinted++
	}
	eventsString += "\n"
	eventsString += quotas

	if eventsLenLimitReached {
		eventsString += tailString
	}

	if throttled {
		eventsString += throttleMsg
	}

	return eventsString
}
