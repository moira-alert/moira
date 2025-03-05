package webhook

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database"
	"github.com/moira-alert/moira/logging"
	"github.com/moira-alert/moira/metrics"
	"github.com/moira-alert/moira/templating"
	"github.com/moira-alert/moira/worker"
)

const (
	webhookDeliveryCheckLockKeyPrefix = "moira-webhook-delivery-check-lock:"
	webhookDeliveryCheckLockTTL       = 30 * time.Second
	workerName                        = "WebhookDeliveryChecker"
)

const (
	logFieldNameDeliveryCheckUrl             = "delivery.check.url"
	logFieldNameDeliveryCheckResponseCode    = "delivery.check.response.code"
	logFieldNameDeliveryCheckResponseBody    = "delivery.check.response.body"
	logFieldNameDeliveryCheckUnknownState    = "delivery.check.unknown.state"
	logFieldNameSendNotificationResponseBody = "send.notification.response.body"
)

type deliveryCheckData struct {
	// Timestamp at which it was scheduled.
	Timestamp int64 `json:"timestamp"`
	// URL for delivery check request.
	URL string `json:"url"`
	// Contact related to delivery check.
	Contact moira.ContactData `json:"contact"`
	// TriggerID for which notification was generated.
	TriggerID string `json:"trigger_id"`
	// AttemptsCount of performing delivery check.
	AttemptsCount uint64 `json:"attempts_count"`
}

func webhookLockKey(contactType string) string {
	return webhookDeliveryCheckLockKeyPrefix + contactType
}

func (sender *Sender) runDeliveryCheckWorker() {
	worker.NewWorker(
		workerName,
		sender.log,
		sender.Database.NewLock(webhookLockKey(sender.contactType), webhookDeliveryCheckLockTTL),
		sender.deliveryCheckerAction,
	).Run(nil)
}

func (sender *Sender) deliveryCheckerAction(stop <-chan struct{}) error {
	checkTicker := time.NewTicker(time.Duration(sender.deliveryCheckConfig.CheckTimeout) * time.Second)

	sender.log.Info().Msg(workerName + " started")
	for {
		select {
		case <-stop:
			sender.log.Info().Msg(workerName + " stopped")
			checkTicker.Stop()
			return nil

		case <-checkTicker.C:
			if err := sender.checkNotificationsDelivery(); err != nil {
				sender.log.Error().
					Error(err).
					Msg("failed to perform delivery check")
			}
		}
	}
}

func (sender *Sender) checkNotificationsDelivery() error {
	fetchTimestamp := sender.clock.NowUnix()

	marshaledData, err := sender.Database.GetDeliveryChecksData(sender.contactType, "-inf", strconv.FormatInt(fetchTimestamp, 10))
	if err != nil {
		if errors.Is(err, database.ErrNil) {
			// nothing to check
			return nil
		}

		return err
	}

	checksData := unmarshalChecksData(sender.log, marshaledData)
	if len(checksData) == 0 {
		return nil
	}

	checksData = removeDuplicatedChecksData(checksData)

	checkAgainChecksData := make([]deliveryCheckData, 0)
	counter := deliveryTypesCounter{}

	for i := range checksData {
		prevDeliveryStopped := counter.deliveryStopped

		var deliveryState string
		checksData[i], deliveryState = sender.performSingleDeliveryCheck(checksData[i])

		newCheckData, scheduleAgain := handleStateTransition(checksData[i], deliveryState, sender.deliveryCheckConfig.MaxAttemptsCount, &counter)
		if scheduleAgain {
			checkAgainChecksData = append(checkAgainChecksData, newCheckData)
		}

		if prevDeliveryStopped != counter.deliveryStopped {
			addContactFieldsToLog(sender.log.Error(), checksData[i].Contact).
				String(logFieldNameDeliveryCheckUrl, checksData[i].URL).
				String(moira.LogFieldNameTriggerID, checksData[i].TriggerID).
				Msg("stop delivery checks")
		}
	}

	err = sender.addDeliveryChecks(checkAgainChecksData, sender.clock.NowUnix()+int64(sender.deliveryCheckConfig.ReschedulingDelay))
	if err != nil {
		return fmt.Errorf("failed to reschedule delivery checks: %w", err)
	}

	err = sender.removeOutdatedDeliveryChecks(fetchTimestamp)
	if err != nil {
		sender.log.Warning().
			Error(err).
			Msg("failed to remove outdated delivery checks")
	}

	markMetrics(sender.metrics, &counter)

	return nil
}

func unmarshalChecksData(logger moira.Logger, marshaledData []string) []deliveryCheckData {
	checksData := make([]deliveryCheckData, 0, len(marshaledData))

	for _, encoded := range marshaledData {
		var data deliveryCheckData
		err := json.Unmarshal([]byte(encoded), &data)
		if err != nil {
			logger.Warning().
				String("encoded_data", encoded).
				Error(err).
				Msg("failed to unmarshal encoded data")
			continue
		}

		checksData = append(checksData, data)
	}

	return checksData
}

func removeDuplicatedChecksData(checksData []deliveryCheckData) []deliveryCheckData {
	deduplicated := make([]deliveryCheckData, 0, len(checksData))
	uniqueMap := make(map[string]deliveryCheckData, len(checksData))

	for _, checkData := range checksData {
		key := fmt.Sprintf("%s_%s_%s_%s", checkData.URL, checkData.Contact.ID, checkData.Contact.Value, checkData.TriggerID)
		if oldCheckData, ok := uniqueMap[key]; ok {
			if oldCheckData.AttemptsCount < checkData.AttemptsCount {
				uniqueMap[key] = checkData
			}
		} else {
			uniqueMap[key] = checkData
		}
	}

	for _, checkData := range uniqueMap {
		deduplicated = append(deduplicated, checkData)
	}

	slices.SortFunc(deduplicated, func(first, second deliveryCheckData) int {
		return int(first.Timestamp - second.Timestamp)
	})

	return deduplicated
}

func (sender *Sender) addDeliveryChecks(checksData []deliveryCheckData, timestamp int64) error {
	if len(checksData) == 0 {
		return nil
	}

	for _, data := range checksData {
		data.Timestamp = timestamp

		encoded, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("failed to marshal check data: %w", err)
		}

		// TODO: retry operations?
		err = sender.Database.AddDeliveryChecksData(sender.contactType, timestamp, string(encoded))
		if err != nil {
			return fmt.Errorf("failed to store check data: %w", err)
		}
	}

	return nil
}

func (sender *Sender) removeOutdatedDeliveryChecks(lastFetchTimestamp int64) error {
	_, err := sender.Database.RemoveDeliveryChecksData(sender.contactType, "-inf", strconv.FormatInt(lastFetchTimestamp, 10))
	return err
}

type deliveryTypesCounter struct {
	deliveryOK      int64
	deliveryFailed  int64
	deliveryStopped int64
}

func (sender *Sender) performSingleDeliveryCheck(checkData deliveryCheckData) (deliveryCheckData, string) {
	var deliveryState string
	checkData.AttemptsCount += 1

	rspCode, rspBody, err := sender.doDeliveryCheckRequest(checkData)
	if err != nil {
		addDeliveryCheckFieldsToLog(
			sender.log.Error().Error(err),
			checkData.URL, rspCode, string(rspBody), checkData.Contact, checkData.TriggerID).
			Msg("check request failed")

		return checkData, moira.DeliveryStateException
	}

	if !isAllowedResponseCode(rspCode) {
		addDeliveryCheckFieldsToLog(
			sender.log.Error().Error(err),
			checkData.URL, rspCode, string(rspBody), checkData.Contact, checkData.TriggerID).
			Msg("not allowed response code")

		return checkData, moira.DeliveryStateException
	}

	var unmarshalledBody map[string]interface{}
	err = json.Unmarshal(rspBody, &unmarshalledBody)
	if err != nil {
		addDeliveryCheckFieldsToLog(
			sender.log.Error().Error(err),
			checkData.URL, rspCode, string(rspBody), checkData.Contact, checkData.TriggerID).
			Msg("failed to unmarshal response")

		return checkData, moira.DeliveryStateException
	}

	populater := templating.NewWebhookDeliveryCheckPopulater(
		&templating.Contact{
			Type:  checkData.Contact.Type,
			Value: checkData.Contact.Value,
		},
		unmarshalledBody,
		checkData.TriggerID)

	deliveryState, err = populater.Populate(sender.deliveryCheckConfig.CheckTemplate)
	if err != nil {
		addDeliveryCheckFieldsToLog(
			sender.log.Error().Error(err),
			checkData.URL, rspCode, string(rspBody), checkData.Contact, checkData.TriggerID).
			Msg("error while populating check template")

		return checkData, moira.DeliveryStateUserException
	}

	if _, ok := moira.DeliveryStatesSet[deliveryState]; !ok {
		addDeliveryCheckFieldsToLog(
			sender.log.Error().String(logFieldNameDeliveryCheckUnknownState, deliveryState),
			checkData.URL, rspCode, string(rspBody), checkData.Contact, checkData.TriggerID).
			Msg("check template returned unknown delivery state")
	}

	return checkData, deliveryState
}

func addDeliveryCheckFieldsToLog(eventBuilder logging.EventBuilder, url string, rspCode int, body string, contact moira.ContactData, triggerID string) logging.EventBuilder {
	return addContactFieldsToLog(eventBuilder, contact).
		String(logFieldNameDeliveryCheckUrl, url).
		Int(logFieldNameDeliveryCheckResponseCode, rspCode).
		String(logFieldNameDeliveryCheckResponseBody, body).
		String(moira.LogFieldNameTriggerID, triggerID)
}

func handleStateTransition(checkData deliveryCheckData, newState string, maxAttemptsCount uint64, counter *deliveryTypesCounter) (deliveryCheckData, bool) {
	switch newState {
	case moira.DeliveryStateOK:
		counter.deliveryOK += 1
		return deliveryCheckData{}, false
	case moira.DeliveryStatePending, moira.DeliveryStateException:
		if checkData.AttemptsCount < maxAttemptsCount {
			return checkData, true
		}

		counter.deliveryStopped += 1
		return deliveryCheckData{}, false
	case moira.DeliveryStateFailed:
		counter.deliveryFailed += 1
		return checkData, false
	case moira.DeliveryStateUserException:
		counter.deliveryStopped += 1
		return deliveryCheckData{}, false
	default:
		counter.deliveryStopped += 1
		return deliveryCheckData{}, false
	}
}

func markMetrics(senderMetrics *metrics.SenderMetrics, counter *deliveryTypesCounter) {
	if senderMetrics == nil || counter == nil {
		return
	}

	senderMetrics.ContactDeliveryNotificationOK.Mark(counter.deliveryOK)
	senderMetrics.ContactDeliveryNotificationFailed.Mark(counter.deliveryFailed)
	senderMetrics.ContactDeliveryNotificationCheckStopped.Mark(counter.deliveryStopped)
}

func (sender *Sender) scheduleDeliveryCheck(contact moira.ContactData, triggerID string, responseBody []byte) {
	var rspData map[string]interface{}
	err := json.Unmarshal(responseBody, &rspData)
	if err != nil {
		addContactFieldsToLog(
			sender.log.Error().Error(err),
			contact).
			String(logFieldNameSendNotificationResponseBody, string(responseBody)).
			String(moira.LogFieldNameTriggerID, triggerID).
			Msg("failed to schedule delivery check because of not unmarshalling")
		return
	}

	checkData, err := prepareDeliveryCheck(contact, rspData, sender.deliveryCheckConfig.URLTemplate, triggerID)
	if err != nil {
		addContactFieldsToLog(
			sender.log.Error().Error(err),
			contact).
			String(logFieldNameSendNotificationResponseBody, string(responseBody)).
			String(moira.LogFieldNameTriggerID, triggerID).
			Msg("failed to prepare delivery check")
		return
	}

	err = sender.addDeliveryChecks([]deliveryCheckData{checkData}, sender.clock.NowUnix())
	if err != nil {
		addContactFieldsToLog(
			sender.log.Error().Error(err),
			contact).
			String(logFieldNameDeliveryCheckUrl, checkData.URL).
			String(logFieldNameSendNotificationResponseBody, string(responseBody)).
			String(moira.LogFieldNameTriggerID, triggerID).
			Msg("failed to prepare delivery check")
		return
	}
}

func prepareDeliveryCheck(contact moira.ContactData, rsp map[string]interface{}, urlTemplate string, triggerID string) (deliveryCheckData, error) {
	urlPopulater := templating.NewWebhookDeliveryCheckURLPopulater(
		&templating.Contact{
			Type:  contact.Type,
			Value: contact.Value,
		},
		rsp,
		triggerID)

	requestURL, err := urlPopulater.Populate(urlTemplate)
	if err != nil {
		return deliveryCheckData{}, fmt.Errorf("failed to fill url template with data: %w", err)
	}

	if err = validateURL(requestURL); err != nil {
		return deliveryCheckData{}, fmt.Errorf("got bad url for check request: %w, url: %s", err, requestURL)
	}

	return deliveryCheckData{
		URL:           requestURL,
		Contact:       contact,
		TriggerID:     triggerID,
		AttemptsCount: 0,
	}, nil
}

func validateURL(requestURL string) error {
	urlStruct, err := url.Parse(requestURL)
	if err != nil {
		return err
	}

	if !(urlStruct.Scheme == "http" || urlStruct.Scheme == "https") {
		return fmt.Errorf("bad url scheme: %s", urlStruct.Scheme)
	}

	if urlStruct.Host == "" {
		return fmt.Errorf("host is empty")
	}

	return nil
}
