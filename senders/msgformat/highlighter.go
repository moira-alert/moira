package msgformat

import (
	"fmt"
	"strings"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/senders"
	"github.com/moira-alert/moira/senders/emoji_provider"
)

// UriFormatter is used for formatting uris, for example for Markdown use something like
// fmt.Sprintf("[%s](%s)", triggerName, triggerURI).
type UriFormatter func(triggerURI, triggerName string) string

// DescriptionFormatter is used to format trigger description to supported description.
type DescriptionFormatter func(trigger moira.TriggerData) string

// BoldFormatter makes str bold. For example in Markdown it should return **str**.
type BoldFormatter func(str string) string

// EventStringFormatter formats single event string.
type EventStringFormatter func(event moira.NotificationEvent, location *time.Location) string

// HighlightSyntaxFormatter formats message by using functions, emojis and some other highlight patterns.
type HighlightSyntaxFormatter struct {
	// emojiGetter used in titles for better description.
	emojiGetter           emoji_provider.StateEmojiGetter
	frontURI              string
	location              *time.Location
	useEmoji              bool
	uriFormatter          UriFormatter
	descriptionFormatter  DescriptionFormatter
	boldFormatter         BoldFormatter
	eventsStringFormatter EventStringFormatter
	codeBlockStart        string
	codeBlockEnd          string
}

// NewHighlightSyntaxFormatter creates new HighlightSyntaxFormatter with given arguments.
func NewHighlightSyntaxFormatter(
	emojiGetter emoji_provider.StateEmojiGetter,
	useEmoji bool,
	frontURI string,
	location *time.Location,
	uriFormatter UriFormatter,
	descriptionFormatter DescriptionFormatter,
	boldFormatter BoldFormatter,
	eventsStringFormatter EventStringFormatter,
	codeBlockStart string,
	codeBlockEnd string,
) MessageFormatter {
	return &HighlightSyntaxFormatter{
		emojiGetter:           emojiGetter,
		frontURI:              frontURI,
		location:              location,
		useEmoji:              useEmoji,
		uriFormatter:          uriFormatter,
		descriptionFormatter:  descriptionFormatter,
		boldFormatter:         boldFormatter,
		eventsStringFormatter: eventsStringFormatter,
		codeBlockStart:        codeBlockStart,
		codeBlockEnd:          codeBlockEnd,
	}
}

// Format formats message using given params and formatter functions.
func (formatter *HighlightSyntaxFormatter) Format(params MessageFormatterParams) string {
	var message strings.Builder
	state := params.Events.GetCurrentState(params.Throttled)
	emoji := formatter.emojiGetter.GetStateEmoji(state)

	title := formatter.buildTitle(params.Events, params.Trigger, emoji, params.Throttled)
	titleLen := len([]rune(title))

	desc := formatter.descriptionFormatter(params.Trigger)
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

func (formatter *HighlightSyntaxFormatter) buildTitle(events moira.NotificationEvents, trigger moira.TriggerData, emoji string, throttled bool) string {
	state := events.GetCurrentState(throttled)
	title := ""
	if formatter.useEmoji {
		title += emoji + " "
	}

	title += formatter.boldFormatter(string(state))
	triggerURI := trigger.GetTriggerURI(formatter.frontURI)
	if triggerURI != "" {
		title += fmt.Sprintf(" %s", formatter.uriFormatter(triggerURI, trigger.Name))
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
func (formatter *HighlightSyntaxFormatter) buildEventsString(events moira.NotificationEvents, charsForEvents int, throttled bool) string {
	charsForThrottleMsg := 0
	throttleMsg := fmt.Sprintf("\nPlease, %s to generate less events.", formatter.boldFormatter(changeRecommendation))
	if throttled {
		charsForThrottleMsg = len([]rune(throttleMsg))
	}
	charsLeftForEvents := charsForEvents - charsForThrottleMsg

	var eventsString string
	eventsString += formatter.codeBlockStart
	var tailString string

	eventsLenLimitReached := false
	eventsPrinted := 0
	for _, event := range events {
		line := fmt.Sprintf("\n%s", formatter.eventsStringFormatter(event, formatter.location))
		if msg := event.CreateMessage(formatter.location); len(msg) > 0 {
			line += fmt.Sprintf(". %s", msg)
		}

		tailString = fmt.Sprintf("\n...and %d more events.", len(events)-eventsPrinted)
		tailStringLen := len([]rune(formatter.codeBlockEnd)) + len("\n") + len([]rune(tailString))
		if !(charsForEvents < 0) && (len([]rune(eventsString))+len([]rune(line)) > charsLeftForEvents-tailStringLen) {
			eventsLenLimitReached = true
			break
		}

		eventsString += line
		eventsPrinted++
	}
	eventsString += "\n"
	eventsString += formatter.codeBlockEnd

	if eventsLenLimitReached {
		eventsString += tailString
	}

	if throttled {
		eventsString += throttleMsg
	}

	return eventsString
}
