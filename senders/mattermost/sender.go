package mattermost

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/senders"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mitchellh/mapstructure"
)

// Structure that represents the Mattermost configuration in the YAML file
type config struct {
	Name        string `mapstructure:"name"`
	Type        string `mapstructure:"type"`
	Url         string `mapstructure:"url"`
	InsecureTLS bool   `mapstructure:"insecure_tls"`
	APIToken    string `mapstructure:"api_token"`
	FrontURI    string `mapstructure:"front_uri"`
}

// Sender posts messages to Mattermost chat.
// It implements moira.Sender.
// You must call Init method before SendEvents method.
type Sender struct {
	clients map[string]*mattermostClient
}

type mattermostClient struct {
	frontURI string
	logger   moira.Logger
	location *time.Location
	client   Client
}

// Init configures Sender.
func (sender *Sender) Init(opts moira.InitOptions) error {
	var cfg config
	err := mapstructure.Decode(opts.SenderSettings, &cfg)
	if err != nil {
		return fmt.Errorf("failed to decode senderSettings to mattermost config: %w", err)
	}

	if cfg.Url == "" {
		return fmt.Errorf("can not read Mattermost url from config")
	}

	if cfg.APIToken == "" {
		return fmt.Errorf("can not read Mattermost api_token from config")
	}

	if cfg.FrontURI == "" {
		return fmt.Errorf("can not read Mattermost front_uri from config")
	}

	client := model.NewAPIv4Client(cfg.Url)

	client.HTTPClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: cfg.InsecureTLS,
			},
		},
	}

	client.SetToken(cfg.APIToken)

	mmClient := &mattermostClient{
		client:   client,
		location: opts.Location,
		logger:   opts.Logger,
		frontURI: cfg.FrontURI,
	}

	var senderIdent string
	if cfg.Name != "" {
		senderIdent = cfg.Name
	} else {
		senderIdent = cfg.Type
	}

	if sender.clients == nil {
		sender.clients = make(map[string]*mattermostClient)
	}

	sender.clients[senderIdent] = mmClient

	return nil
}

// SendEvents implements moira.Sender interface.
func (sender *Sender) SendEvents(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plots [][]byte, throttled bool) error {
	client, ok := sender.clients[contact.Type]
	if !ok {
		return fmt.Errorf("failed to send events because there is not %s client", contact.Type)
	}

	message := client.buildMessage(events, trigger, throttled)
	ctx := context.Background()
	post, err := client.sendMessage(ctx, message, contact.Value, trigger.ID)
	if err != nil {
		return err
	}

	if len(plots) > 0 {
		err = client.sendPlots(ctx, plots, contact.Value, post.Id, trigger.ID)
		if err != nil {
			client.logger.Warning().
				String("trigger_id", trigger.ID).
				String("contact_value", contact.Value).
				String("contact_type", contact.Type).
				Error(err)
		}
	}

	return nil
}

func (client *mattermostClient) buildMessage(events moira.NotificationEvents, trigger moira.TriggerData, throttled bool) string {
	const messageMaxCharacters = 4_000

	var message strings.Builder

	title := client.buildTitle(events, trigger, throttled)
	titleLen := len([]rune(title))

	desc := client.buildDescription(trigger)
	descLen := len([]rune(desc))

	eventsString := client.buildEventsString(events, -1, throttled)
	eventsStringLen := len([]rune(eventsString))

	charsLeftAfterTitle := messageMaxCharacters - titleLen

	descNewLen, eventsNewLen := senders.CalculateMessagePartsLength(charsLeftAfterTitle, descLen, eventsStringLen)

	if descLen != descNewLen {
		desc = desc[:descNewLen] + "...\n"
	}
	if eventsNewLen != eventsStringLen {
		eventsString = client.buildEventsString(events, eventsNewLen, throttled)
	}

	message.WriteString(title)
	message.WriteString(desc)
	message.WriteString(eventsString)
	return message.String()
}

func (client *mattermostClient) buildDescription(trigger moira.TriggerData) string {
	desc := trigger.Desc
	if trigger.Desc != "" {
		desc += "\n"
	}

	return desc
}

func (client *mattermostClient) buildTitle(events moira.NotificationEvents, trigger moira.TriggerData, throttled bool) string {
	state := events.GetCurrentState(throttled)
	title := fmt.Sprintf("**%s**", state)
	triggerURI := trigger.GetTriggerURI(client.frontURI)
	if triggerURI != "" {
		title += fmt.Sprintf(" [%s](%s)", trigger.Name, triggerURI)
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
// if n is negative buildEventsString does not limit the events string
func (client *mattermostClient) buildEventsString(events moira.NotificationEvents, charsForEvents int, throttled bool) string {
	charsForThrottleMsg := 0
	throttleMsg := "\nPlease, *fix your system or tune this trigger* to generate less events."
	if throttled {
		charsForThrottleMsg = len([]rune(throttleMsg))
	}
	charsLeftForEvents := charsForEvents - charsForThrottleMsg

	var eventsString = "```"
	var tailString string

	eventsLenLimitReached := false
	eventsPrinted := 0
	for _, event := range events {
		line := fmt.Sprintf("\n%s: %s = %s (%s to %s)", event.FormatTimestamp(client.location, moira.DefaultTimeFormat), event.Metric, event.GetMetricsValues(moira.DefaultNotificationSettings), event.OldState, event.State)
		if msg := event.CreateMessage(client.location); len(msg) > 0 {
			line += fmt.Sprintf(". %s", msg)
		}

		tailString = fmt.Sprintf("\n...and %d more events.", len(events)-eventsPrinted)
		tailStringLen := len([]rune("```")) + len([]rune(tailString))
		if !(charsForEvents < 0) && (len([]rune(eventsString))+len([]rune(line)) > charsLeftForEvents-tailStringLen) {
			eventsLenLimitReached = true
			break
		}

		eventsString += line
		eventsPrinted++
	}
	eventsString += "```"

	if eventsLenLimitReached {
		eventsString += tailString
	}

	if throttled {
		eventsString += throttleMsg
	}

	return eventsString
}

func (client *mattermostClient) sendMessage(ctx context.Context, message string, contact string, triggerID string) (*model.Post, error) {
	post := model.Post{
		ChannelId: contact,
		Message:   message,
	}

	sentPost, _, err := client.client.CreatePost(ctx, &post)
	if err != nil {
		return nil, fmt.Errorf("failed to send %s event message to Mattermost [%s]: %s", triggerID, contact, err)
	}

	return sentPost, nil
}

func (client *mattermostClient) sendPlots(ctx context.Context, plots [][]byte, channelID, postID, triggerID string) error {
	var filesID []string

	filename := fmt.Sprintf("%s.png", triggerID)
	for _, plot := range plots {
		file, _, err := client.client.UploadFile(ctx, plot, channelID, filename)
		if err != nil {
			return err
		}
		for _, info := range file.FileInfos {
			filesID = append(filesID, info.Id)
		}
	}

	if len(filesID) > 0 {
		_, _, err := client.client.CreatePost(
			ctx,
			&model.Post{
				ChannelId: channelID,
				RootId:    postID,
				FileIds:   filesID,
			},
		)
		if err != nil {
			return err
		}
	}

	return nil
}
