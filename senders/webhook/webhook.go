package webhook

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/clock"
	"github.com/moira-alert/moira/logging"
	"github.com/moira-alert/moira/metrics"
)

// Structure that represents the Webhook configuration in the YAML file.
type config struct {
	ContactType   string              `mapstructure:"contact_type"`
	URL           string              `mapstructure:"url" validate:"required"`
	Body          string              `mapstructure:"body"`
	Headers       map[string]string   `mapstructure:"headers"`
	User          string              `mapstructure:"user"`
	Password      string              `mapstructure:"password"`
	Timeout       int                 `mapstructure:"timeout"`
	DeliveryCheck deliveryCheckConfig `mapstructure:"delivery_check"`
}

type deliveryCheckConfig struct {
	// Enabled the delivery checking or not.
	Enabled bool `mapstructure:"enabled"`
	// URLTemplate is need to build url for GET HTTP request, used for delivery checking.
	// Template is filled based on contact data and response, got on sending POST request.
	URLTemplate string `mapstructure:"url_template" validate:"required_if=Enabled true"`
	// Headers for delivery check request.
	Headers map[string]string `mapstructure:"headers"`
	// User for delivery check request.
	User string `mapstructure:"user"`
	// Password for delivery check request.
	Password string `mapstructure:"password"`
	// CheckTemplate must calculate the notification delivery state based on the response for delivery. Must return one of:
	//	- moira.DeliveryStateOK
	//	- moira.DeliveryStatePending
	//	- moira.DeliveryStateFailed
	//	- moira.DeliveryStateException
	CheckTemplate string `mapstructure:"check_template" validate:"required_if=Enabled true"`
	// CheckTimeout is the timeout (in seconds) between checking notifications delivery.
	CheckTimeout uint64 `mapstructure:"check_timeout"`
	// MaxAttemptsCount will be performed to understand if the notification was delivered or not.
	// After that delivery checks will stop.
	MaxAttemptsCount uint64 `mapstructure:"max_attempts_count"`
	// ReschedulingDelay is added to the clock.NowUnix() than schedule next check attempt.
	ReschedulingDelay uint64 `mapstructure:"rescheduling_delay"`
}

const (
	defaultCheckTimeout      = 60
	defaultMaxAttemptsCount  = 5
	defaultReschedulingDelay = 45
)

// Sender implements moira sender interface via webhook.
type Sender struct {
	url            string
	body           string
	user           string
	password       string
	headers        map[string]string
	contactType    string
	client         *http.Client
	log            moira.Logger
	metrics        *metrics.SenderMetrics
	Database       moira.DeliveryCheckerDatabase
	deliveryConfig deliveryCheckConfig
	clock          moira.Clock
}

func getDefaultDeliveryCheckConfig() deliveryCheckConfig {
	return deliveryCheckConfig{
		Enabled:           false,
		CheckTimeout:      defaultCheckTimeout,
		MaxAttemptsCount:  defaultMaxAttemptsCount,
		ReschedulingDelay: defaultReschedulingDelay,
	}
}

const senderMetricsKey = "sender_metrics"

// Init read yaml config.
func (sender *Sender) Init(senderSettings interface{}, logger moira.Logger, location *time.Location, dateTimeFormat string) error {
	var cfg config
	cfg.DeliveryCheck = getDefaultDeliveryCheckConfig()

	err := mapstructure.Decode(senderSettings, &cfg)
	if err != nil {
		return fmt.Errorf("failed to decode senderSettings to webhook config: %w", err)
	}

	if err = moira.ValidateStruct(cfg); err != nil {
		return fmt.Errorf("webhook config validation error: %w", err)
	}

	sender.url = cfg.URL
	sender.body = cfg.Body
	sender.user, sender.password = cfg.User, cfg.Password

	sender.headers = map[string]string{
		"User-Agent":   "Moira",
		"Content-Type": "application/json",
	}

	for header, value := range cfg.Headers {
		sender.headers[header] = value
	}

	var timeout int
	if cfg.Timeout != 0 {
		timeout = cfg.Timeout
	} else {
		timeout = 30
	}

	sender.log = logger
	sender.client = &http.Client{
		Timeout:   time.Duration(timeout) * time.Second,
		Transport: &http.Transport{DisableKeepAlives: true},
	}

	senderSettingsMap := senderSettings.(map[string]interface{})
	if val, ok := senderSettingsMap[senderMetricsKey]; ok {
		sender.metrics = val.(*metrics.SenderMetrics)
	}

	sender.deliveryConfig = cfg.DeliveryCheck
	sender.clock = clock.NewSystemClock()
	if sender.deliveryConfig.Enabled {
		if sender.metrics == nil {
			return fmt.Errorf("with enabled delivery check, sender must have 'enable_metrcis: true'")
		}

		go sender.runDeliveryCheckWorker()
	}

	return nil
}

// SendEvents implements Sender interface Send.
func (sender *Sender) SendEvents(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plots [][]byte, throttled bool) error {
	request, err := sender.buildSendAlertRequest(events, contact, trigger, plots, throttled)
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}
	defer request.Body.Close()

	responseStatusCode, responseBody, err := performRequest(sender.client, request)
	if err != nil {
		return fmt.Errorf("send alert request failed: %w", err)
	}

	if !isAllowedResponseCode(responseStatusCode) {
		return fmt.Errorf("invalid status code: %d, server response: %s", responseStatusCode, string(responseBody))
	}

	if sender.deliveryConfig.Enabled {
		var rspData map[string]interface{}
		err = json.Unmarshal(responseBody, &rspData)
		if err != nil {
			addContactFieldsToLog(
				sender.log.Error().Error(err),
				contact).
				String(logFieldNameSendNotificationResponseBody, string(responseBody)).
				Msg("failed to schedule delivery check because of not unmarshalling")
			return nil
		}

		checkData, err := prepareDeliveryCheck(contact, rspData, sender.deliveryConfig.URLTemplate)
		if err != nil {
			addContactFieldsToLog(
				sender.log.Error().Error(err),
				contact).
				String(logFieldNameSendNotificationResponseBody, string(responseBody)).
				Msg("failed to prepare delivery check")
			return nil
		}

		err = sender.scheduleDeliveryChecks([]deliveryCheckData{checkData}, sender.clock.NowUnix())
		if err != nil {
			addContactFieldsToLog(
				sender.log.Error().Error(err),
				contact).
				String(logFieldNameDeliveryCheckUrl, checkData.URL).
				String(logFieldNameSendNotificationResponseBody, string(responseBody)).
				Msg("failed to prepare delivery check")
			return nil
		}
	}

	return nil
}

func isAllowedResponseCode(responseCode int) bool {
	return (responseCode >= http.StatusOK) && (responseCode < http.StatusMultipleChoices)
}

func addContactFieldsToLog(eventBuilder logging.EventBuilder, contact moira.ContactData) logging.EventBuilder {
	return eventBuilder.
		String(moira.LogFieldNameContactID, contact.ID).
		String(moira.LogFieldNameContactType, contact.Type).
		String(moira.LogFieldNameContactValue, contact.Value)
}
