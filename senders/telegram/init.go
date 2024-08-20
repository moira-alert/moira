package telegram

import (
	"errors"
	"fmt"
	"github.com/russross/blackfriday/v2"
	"html"
	"regexp"
	"strings"
	"time"

	"github.com/moira-alert/moira/senders/msgformat"

	"github.com/mitchellh/mapstructure"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/worker"
	"gopkg.in/telebot.v3"
)

const (
	telegramLockPrefix = "moira-telegram-users:moira-bot-host:"
	workerName         = "Telebot"
	messenger          = "telegram"
	telegramLockTTL    = 30 * time.Second
	hidden             = "[DATA DELETED]"
)

var (
	startHeaderRegexp = regexp.MustCompile("<h[0-9]+>")
	endHeaderRegexp   = regexp.MustCompile("</h[0-9]+>")
	codeBlockStart    = "<blockquote expandable>"
	codeBlockEnd      = "</blockquote>"
)

var pollerTimeout = 10 * time.Second

// Structure that represents the Telegram configuration in the YAML file.
type config struct {
	ContactType string `mapstructure:"contact_type"`
	APIToken    string `mapstructure:"api_token"`
	FrontURI    string `mapstructure:"front_uri"`
}

// Bot is abstraction over gopkg.in/telebot.v3#Bot.
type Bot interface {
	Handle(endpoint interface{}, h telebot.HandlerFunc, m ...telebot.MiddlewareFunc)
	Start()
	Stop()
	Send(to telebot.Recipient, what interface{}, opts ...interface{}) (*telebot.Message, error)
	SendAlbum(to telebot.Recipient, a telebot.Album, opts ...interface{}) ([]telebot.Message, error)
	Reply(to *telebot.Message, what interface{}, opts ...interface{}) (*telebot.Message, error)
	ChatByUsername(name string) (*telebot.Chat, error)
}

// Sender implements moira sender interface via telegram.
type Sender struct {
	DataBase  moira.Database
	logger    moira.Logger
	bot       Bot
	formatter msgformat.MessageFormatter
	apiToken  string
}

func (sender *Sender) removeTokenFromError(err error) error {
	if err != nil && strings.Contains(err.Error(), sender.apiToken) {
		return errors.New(strings.Replace(err.Error(), sender.apiToken, hidden, -1))
	}
	return err
}

// Init loads yaml config, configures and starts telegram bot.
func (sender *Sender) Init(senderSettings interface{}, logger moira.Logger, location *time.Location, dateTimeFormat string) error {
	var cfg config
	err := mapstructure.Decode(senderSettings, &cfg)
	if err != nil {
		return fmt.Errorf("failed to decode senderSettings to telegram config: %w", err)
	}

	if cfg.APIToken == "" {
		return fmt.Errorf("can not read telegram api_token from config")
	}
	sender.apiToken = cfg.APIToken

	emojiProvider := telegramEmojiProvider{}
	sender.formatter = msgformat.NewHighlightSyntaxFormatter(
		emojiProvider,
		true,
		cfg.FrontURI,
		location,
		urlFormatter,
		descriptionFormatter,
		boldFormatter,
		eventStringFormatter,
		codeBlockStart,
		codeBlockEnd)

	sender.logger = logger
	sender.bot, err = telebot.NewBot(telebot.Settings{
		Token:  cfg.APIToken,
		Poller: &telebot.LongPoller{Timeout: pollerTimeout},
	})
	if err != nil {
		return sender.removeTokenFromError(err)
	}

	sender.bot.Handle(telebot.OnText, func(ctx telebot.Context) error {
		if err = sender.handleMessage(ctx.Message()); err != nil {
			sender.logger.Error().
				Error(err).
				Msg("Error handling incoming message")
			return err
		}
		return nil
	})

	go sender.runTelebot(cfg.ContactType)

	return nil
}

// runTelebot starts telegram bot and manages bot subscriptions
// to make sure there is always only one working Poller.
func (sender *Sender) runTelebot(contactType string) {
	workerAction := func(stop <-chan struct{}) error {
		sender.bot.Start()
		<-stop
		sender.bot.Stop()
		return nil
	}

	worker.NewWorker(
		workerName,
		sender.logger,
		sender.DataBase.NewLock(telegramLockKey(contactType), telegramLockTTL),
		workerAction,
	).Run(nil)
}

func telegramLockKey(contactType string) string {
	return telegramLockPrefix + contactType
}

func urlFormatter(triggerURI, triggerName string) string {
	return fmt.Sprintf("<a href=\"%s\">%s</a>", triggerURI, html.EscapeString(triggerName))
}

const (
	endSuffix = "...\n"
)

func descriptionFormatter(trigger moira.TriggerData, maxSize int) string {
	if maxSize == 0 {
		return ""
	}

	desc := trigger.Desc
	if trigger.Desc != "" {
		desc = trigger.Desc
		desc += "\n"
	}

	htmlDescStr := string(blackfriday.Run([]byte(desc)))

	// html headers are not supported by telegram html, so make them bold instead.
	htmlDescStr = startHeaderRegexp.ReplaceAllString(htmlDescStr, "<b>")
	replacedHeaders := endHeaderRegexp.ReplaceAllString(htmlDescStr, "</b>")

	// some tags are not supported or too long, so replace them.
	replacer := strings.NewReplacer(
		"<p>", "",
		"</p>", "",
		"&ldquo;", "&quot;",
		"&rdquo;", "&quot;",
		"<strong>", "<b>",
		"</strong>", "</b>",
		"<em>", "<i>",
		"</em>", "</i>",
		"<del>", "<s>",
		"</del>", "</s>")
	withReplacedTags := replacer.Replace(replacedHeaders)

	descRunes := []rune(withReplacedTags)
	if maxSize < 0 || len(descRunes) <= maxSize {
		return withReplacedTags
	}

	return cutDescription(descRunes, maxSize-len(endSuffix)) + endSuffix
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

type descriptionNodeType int8

const (
	undefined     descriptionNodeType = iota // if node type is undefined.
	openTag                                  // for example <b>.
	closeTag                                 // for example </b>.
	text                                     // text with no tags or escaped symbols.
	escapedSymbol                            // escaped symbols, for example '>' is turned into '&gt;'.
)

type descriptionNode struct {
	content []rune
	// start of content in the description
	start    int
	nodeType descriptionNodeType
}

func splitDescriptionIntoNodes(fullDesc []rune, maxSize int) ([]descriptionNode, []int, int) {
	var nodes []descriptionNode
	var stack []int

	var nodeContent []rune
	prevNodeType := undefined
	startOfNode := 0
	for i := 0; i < maxSize; i++ {
		r := fullDesc[i]

		// tagStarted
		if r == '<' {
			if len(nodeContent) != 0 {
				nodes = append(nodes, descriptionNode{
					content:  nodeContent,
					start:    startOfNode,
					nodeType: prevNodeType,
				})
				nodeContent = []rune{}
			}
			prevNodeType = openTag
			startOfNode = i
		}

		if len(nodeContent) == 1 && nodeContent[0] == '<' && r == '/' {
			prevNodeType = closeTag
		}

		// start of escaped symbol
		if r == '&' && len(nodeContent) != 0 {
			if len(nodeContent) != 0 {
				nodes = append(nodes, descriptionNode{
					content:  nodeContent,
					start:    startOfNode,
					nodeType: prevNodeType,
				})
				nodeContent = []rune{}
			}
			prevNodeType = escapedSymbol
			startOfNode = i
		}

		nodeContent = append(nodeContent, r)

		// tag ended
		if r == '>' {
			nodes = append(nodes, descriptionNode{
				content:  nodeContent,
				start:    startOfNode,
				nodeType: prevNodeType,
			})

			if prevNodeType == openTag {
				stack = append(stack, len(nodes)-1)
			} else if prevNodeType == closeTag {
				stack = stack[:len(stack)-1]
			}

			nodeContent = []rune{}
			prevNodeType = undefined
			startOfNode = i + 1
			continue
		}

		// end of escaped symbol
		if len(nodeContent) > 0 && nodeContent[0] == '&' && r == ';' {
			nodes = append(nodes, descriptionNode{
				content:  nodeContent,
				start:    startOfNode,
				nodeType: prevNodeType,
			})
			nodeContent = []rune{}
			prevNodeType = undefined
			startOfNode = i + 1
			continue
		}

		// not the tag nor escaped symbol
		if len(nodeContent) == 1 {
			prevNodeType = text
			startOfNode = i
		}
	}

	if len(nodeContent) != 0 {
		if nodeContent[0] == '<' || nodeContent[0] == '&' {
			maxSize -= len(nodeContent)
		} else {
			nodes = append(nodes, descriptionNode{
				content:  nodeContent,
				start:    startOfNode,
				nodeType: prevNodeType,
			})
		}
	}

	return nodes, stack, maxSize
}

func toString(nodes []descriptionNode) string {
	var res string

	for _, node := range nodes {
		res += string(node.content)
	}

	return res
}

func lenContent(nodes []descriptionNode) int {
	res := 0

	for _, node := range nodes {
		res += len(node.content)
	}

	return res
}

func cutDescription(fullDesc []rune, maxSize int) string {
	var nodes []descriptionNode
	var unclosed []int

	nodes, unclosed, maxSize = splitDescriptionIntoNodes(fullDesc, maxSize)

	if len(unclosed) == 0 {
		return toString(nodes)
	}

	var reversedCloseTags []descriptionNode

	for i, nodeIdx := range unclosed {
		if string(nodes[nodeIdx].content) == "<pre>" {
			// remove <pre> block

			nodes = nodes[:nodeIdx]
			unclosed = unclosed[:i]
			break
		} else if strings.HasPrefix(string(nodes[nodeIdx].content), "<a href=\"") {
			// try to save link, but if there is no space for closing all tags remove it and try to close other tags

			var noTagsNodes []descriptionNode
			skippedLen := 0
			textLen := 0

			for j := nodeIdx + 1; j < len(nodes); j++ {
				if nodes[j].nodeType == text || nodes[j].nodeType == escapedSymbol {
					if len(noTagsNodes) > 0 && noTagsNodes[len(noTagsNodes)-1].nodeType == nodes[j].nodeType {
						// merge nodes with the same types

						noTagsNodes[len(noTagsNodes)-1].content = append(noTagsNodes[len(noTagsNodes)-1].content, nodes[j].content...)
					} else {
						noTagsNodes = append(noTagsNodes, nodes[j])
					}

					textLen += len(nodes[j].content)
				} else {
					skippedLen += len(nodes[j].content)
				}
			}

			linkCloseRunes := []rune("</a>")
			lenReversedTags := lenContent(reversedCloseTags)
			if skippedLen+textLen > len(linkCloseRunes)+lenReversedTags {
				// we have enough space to close all tags and left the link

				if skippedLen < len(linkCloseRunes)+lenReversedTags {
					// we have enough space to close all tags, but we have to cut text inside of link

					newTextLen := skippedLen + textLen - len(linkCloseRunes) - lenReversedTags

					// cutting text
					cutLen := 0
					for noTagsNodesIdx := 0; noTagsNodesIdx < len(noTagsNodes); noTagsNodesIdx++ {
						if cutLen+len(noTagsNodes[noTagsNodesIdx].content) < newTextLen {
							cutLen += len(noTagsNodes[noTagsNodesIdx].content)
						} else if cutLen+len(noTagsNodes[noTagsNodesIdx].content) == newTextLen {
							cutLen += len(noTagsNodes[noTagsNodesIdx].content)
							noTagsNodes = noTagsNodes[:noTagsNodesIdx+1]
							break
						} else {
							if noTagsNodes[noTagsNodesIdx].nodeType == escapedSymbol {
								noTagsNodes = noTagsNodes[:noTagsNodesIdx]
								break
							} else {
								noTagsNodes[noTagsNodesIdx].content = noTagsNodes[noTagsNodesIdx].content[:newTextLen-cutLen]
								noTagsNodes = noTagsNodes[:noTagsNodesIdx+1]
								break
							}
						}
					}
				}

				nodes = append(nodes[:nodeIdx+1], noTagsNodes...)
				nodes = append(nodes, descriptionNode{
					content:  linkCloseRunes,
					nodeType: closeTag,
				})
				nodes = append(nodes, reversedCloseTags...)
				unclosed = []int{}
				break
			} else {
				// there is no enough space for link, so remove it

				nodes = nodes[:nodeIdx]
				unclosed = unclosed[:i]
				break
			}
		} else {
			// if we have such unclosed tags: <b>, <i>, <s> then
			// reversedCloseTags should be: </s>, </i>, </b>

			var newContent []rune
			newContent = append(newContent, nodes[nodeIdx].content[0], '/')
			newContent = append(newContent, nodes[nodeIdx].content[1:]...)

			if len(reversedCloseTags) == 0 {
				reversedCloseTags = append(reversedCloseTags, descriptionNode{
					content:  newContent,
					nodeType: closeTag,
				})
			} else {
				tail := reversedCloseTags
				reversedCloseTags = []descriptionNode{}
				reversedCloseTags = append(reversedCloseTags, descriptionNode{
					content:  newContent,
					nodeType: closeTag,
				})
				reversedCloseTags = append(reversedCloseTags, tail...)
			}
		}
	}

	if len(unclosed) == 0 {
		return toString(nodes)
	}

	currentLen := lenContent(nodes)
	lenCloseTags := lenContent(reversedCloseTags)

	if maxSize < currentLen+lenCloseTags {
		for i := len(nodes) - 1; i >= 0; i-- {
			switch nodes[i].nodeType {
			case escapedSymbol:
				currentLen = lenContent(nodes[:i])
			case text:
				toCutLen := currentLen + lenCloseTags - maxSize

				// if we can cut text to give us space for
				if toCutLen >= len(nodes[i].content) {
					currentLen = lenContent(nodes[:i])
				} else {
					nodes[i].content = nodes[i].content[:len(nodes[i].content)-toCutLen]
					nodes = nodes[:i+1]
					break
				}
			case closeTag:
				tail := reversedCloseTags
				reversedCloseTags = []descriptionNode{}
				reversedCloseTags = append(reversedCloseTags, descriptionNode{
					content:  nodes[i].content,
					nodeType: closeTag,
				})
				reversedCloseTags = append(reversedCloseTags, tail...)
				currentLen = lenContent(nodes[:i])
				lenCloseTags = lenContent(reversedCloseTags)
			case openTag:
				if len(reversedCloseTags) == 1 {
					nodes = nodes[:i]
					reversedCloseTags = []descriptionNode{}
					break
				}
				reversedCloseTags = reversedCloseTags[1:]
				lenCloseTags = lenContent(reversedCloseTags)
				currentLen = lenContent(nodes[:i])
			}

			if maxSize >= currentLen+lenCloseTags {
				nodes = nodes[:i]
				break
			}
		}
	}

	nodes = append(nodes, reversedCloseTags...)

	return toString(nodes)
}
