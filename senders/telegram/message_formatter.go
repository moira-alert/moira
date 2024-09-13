package telegram

import (
	"fmt"
	"html"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/senders"
	"github.com/moira-alert/moira/senders/emoji_provider"
	"github.com/moira-alert/moira/senders/msgformat"
	"github.com/russross/blackfriday/v2"
)

const (
	eventsBlockStart = "<blockquote>"
	eventsBlockEnd   = "</blockquote>"
)

type messageFormatter struct {
	// emojiGetter used in titles for better description.
	emojiGetter emoji_provider.StateEmojiGetter
	frontURI    string
	location    *time.Location
	useEmoji    bool
}

// NewTelegramMessageFormatter returns message formatter which is used in telegram sender.
// The message will be formatted with html tags supported by telegram.
func NewTelegramMessageFormatter(
	emojiGetter emoji_provider.StateEmojiGetter,
	useEmoji bool,
	frontURI string,
	location *time.Location,
) msgformat.MessageFormatter {
	return &messageFormatter{
		emojiGetter: emojiGetter,
		frontURI:    frontURI,
		location:    location,
		useEmoji:    useEmoji,
	}
}

// Format formats message using given params and formatter functions.
func (formatter *messageFormatter) Format(params msgformat.MessageFormatterParams) string {
	params.Trigger.Tags = htmlEscapeTags(params.Trigger.Tags)

	state := params.Events.GetCurrentState(params.Throttled)
	emoji := formatter.emojiGetter.GetStateEmoji(state)

	title := formatter.buildTitle(params.Events, params.Trigger, emoji, params.Throttled)
	titleLen := calcRunesCountWithoutHTML(title) + len("\n")

	tagsStr := " " + params.Trigger.GetTags()
	tagsLen := calcRunesCountWithoutHTML(tagsStr)

	if tagsLen == len(" ") {
		tagsStr = ""
		tagsLen = 0
	}

	desc := descriptionFormatter(params.Trigger)
	descLen := calcRunesCountWithoutHTML(desc)

	eventsString := formatter.buildEventsString(params.Events, -1, params.Throttled)
	eventsStringLen := calcRunesCountWithoutHTML(eventsString)

	tagsNewLen, descNewLen, eventsNewLen := senders.CalculateMessageParts(params.MessageMaxChars-titleLen, tagsLen, descLen, eventsStringLen)
	if tagsLen != tagsNewLen {
		tagsStr = msgformat.DefaultTagsLimiter(params.Trigger.Tags, tagsNewLen)
	}
	if descLen != descNewLen {
		desc = descriptionCutter(desc, descNewLen)
	}
	if eventsStringLen != eventsNewLen {
		eventsString = formatter.buildEventsString(params.Events, eventsNewLen, params.Throttled)
	}

	return title + tagsStr + "\n" + desc + eventsString
}

func htmlEscapeTags(tags []string) []string {
	escapedTags := make([]string, 0, len(tags))

	for _, tag := range tags {
		escapedTags = append(escapedTags, html.EscapeString(tag))
	}

	return escapedTags
}

// calcRunesCountWithoutHTML is used for calculating symbols in text without html tags. Special symbols
// like `&gt;`, `&lt;` etc. are counted not as one symbol, for example, len([]rune("&gt;")).
// This precision is enough for us to evaluate size of message.
func calcRunesCountWithoutHTML(htmlText string) int {
	textLen := 0
	isTag := false

	for _, r := range []rune(htmlText) {
		if r == '<' {
			isTag = true
			continue
		}

		if !isTag {
			textLen += 1
		}

		if r == '>' {
			isTag = false
		}
	}

	return textLen
}

func (formatter *messageFormatter) buildTitle(events moira.NotificationEvents, trigger moira.TriggerData, emoji string, throttled bool) string {
	state := events.GetCurrentState(throttled)
	title := ""
	if formatter.useEmoji {
		title += emoji + " "
	}

	title += boldFormatter(string(state))
	triggerURI := trigger.GetTriggerURI(formatter.frontURI)
	if triggerURI != "" {
		title += fmt.Sprintf(" %s", uriFormatter(triggerURI, trigger.Name))
	} else if trigger.Name != "" {
		title += " " + trigger.Name
	}

	return title
}

var throttleMsg = fmt.Sprintf("\nPlease, %s to generate less events.", boldFormatter(msgformat.ChangeTriggerRecommendation))

// buildEventsString builds the string from moira events and limits it to charsForEvents.
// if charsForEvents is negative buildEventsString does not limit the events string.
func (formatter *messageFormatter) buildEventsString(events moira.NotificationEvents, charsForEvents int, throttled bool) string {
	charsForThrottleMsg := 0
	if throttled {
		charsForThrottleMsg = calcRunesCountWithoutHTML(throttleMsg)
	}
	charsLeftForEvents := charsForEvents - charsForThrottleMsg

	var eventsString string
	eventsString += eventsBlockStart
	var tailString string

	eventsLenLimitReached := false
	eventsPrinted := 0
	eventsStringLen := 0
	for _, event := range events {
		line := fmt.Sprintf("\n%s", eventStringFormatter(event, formatter.location))
		if msg := event.CreateMessage(formatter.location); len(msg) > 0 {
			line += fmt.Sprintf(". %s", msg)
		}

		tailString = fmt.Sprintf("\n...and %d more events.", len(events)-eventsPrinted)
		tailStringLen := len("\n") + utf8.RuneCountInString(tailString)
		lineLen := calcRunesCountWithoutHTML(line)

		if charsForEvents >= 0 && eventsStringLen+lineLen > charsLeftForEvents-tailStringLen {
			eventsLenLimitReached = true
			break
		}

		eventsString += line
		eventsStringLen += lineLen
		eventsPrinted++
	}
	eventsString += "\n"
	eventsString += eventsBlockEnd

	if eventsLenLimitReached {
		eventsString += tailString
	}

	if throttled {
		eventsString += throttleMsg
	}

	return eventsString
}

func uriFormatter(triggerURI, triggerName string) string {
	return fmt.Sprintf("<a href=\"%s\">%s</a>", triggerURI, html.EscapeString(triggerName))
}

var (
	startHeaderRegexp = regexp.MustCompile("<h[0-9]+>")
	endHeaderRegexp   = regexp.MustCompile("</h[0-9]+>")
)

func descriptionFormatter(trigger moira.TriggerData) string {
	if trigger.Desc == "" {
		return ""
	}

	desc := trigger.Desc + "\n"

	// Sometimes in trigger description may be text constructions like <param>.
	// blackfriday may recognise it as tag, so it won't be escaped.
	// Then it is sent to telegram we will get error: 'Bad request', because telegram doesn't support such tag.
	// So escaping them before blackfriday.Run.
	replacer := strings.NewReplacer(
		"<", "&lt;",
		">", "&gt;",
	)
	mdWithNoTags := replacer.Replace(desc)

	htmlDescStr := string(blackfriday.Run([]byte(mdWithNoTags),
		blackfriday.WithExtensions(
			blackfriday.CommonExtensions &
				^blackfriday.DefinitionLists &
				^blackfriday.Tables),
		blackfriday.WithRenderer(
			blackfriday.NewHTMLRenderer(
				blackfriday.HTMLRendererParameters{
					Flags: blackfriday.UseXHTML,
				}))))

	// html headers are not supported by telegram html, so make them bold instead.
	htmlDescStr = startHeaderRegexp.ReplaceAllString(htmlDescStr, "<b>")
	replacedHeaders := endHeaderRegexp.ReplaceAllString(htmlDescStr, "</b>")

	// some tags are not supported, so replace them.
	tagReplacer := strings.NewReplacer(
		"<p>", "",
		"</p>", "",
		"<ul>", "",
		"</ul>", "",
		"<li>", "- ",
		"</li>", "",
		"<ol>", "",
		"</ol>", "",
		"<hr>", "",
		"<hr />", "",
		"<br>", "\n",
		"<br />", "\n")

	return tagReplacer.Replace(replacedHeaders)
}

const (
	tooLongDescMessage = "\nDescription is too long for telegram sender.\n"
	badFormatMessage   = "\nBad trigger description for telegram sender. Please check trigger.\n"
)

func descriptionCutter(_ string, maxSize int) string {
	if utf8.RuneCountInString(tooLongDescMessage) <= maxSize {
		return tooLongDescMessage
	}

	return ""
}

func boldFormatter(str string) string {
	return fmt.Sprintf("<b>%s</b>", html.EscapeString(str))
}

func eventStringFormatter(event moira.NotificationEvent, loc *time.Location) string {
	return fmt.Sprintf(
		"%s: <code>%s</code> = %s (%s to %s)",
		event.FormatTimestamp(loc, moira.DefaultTimeFormat),
		html.EscapeString(event.Metric),
		html.EscapeString(event.GetMetricsValues(moira.DefaultNotificationSettings)),
		event.OldState,
		event.State)
}
