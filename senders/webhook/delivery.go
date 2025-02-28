package webhook

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database"
	"github.com/moira-alert/moira/metrics"
	"github.com/moira-alert/moira/templating"
	"github.com/moira-alert/moira/worker"
)

const (
	webhookDeliveryCheckLockKeyPrefix = "moira-webhook-delivery-check-lock:"
	webhookDeliveryCheckLockTTL       = 30 * time.Second
	workerName                        = "WebhookDeliveryChecker"
)

type deliveryCheckData struct {
	URL           string            `json:"url"`
	PreviousState string            `json:"previous_state"`
	Contact       moira.ContactData `json:"contact"`
	AttemptsCount uint64            `json:"attempts_count"`
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
	checkTicker := time.NewTicker(time.Duration(sender.deliveryConfig.CheckTimeout) * time.Second)

	sender.log.Info().Msg(workerName + " started")
	for {
		select {
		case <-stop:
			sender.log.Info().Msg(workerName + " stopped")
			checkTicker.Stop()
			return nil

		case <-checkTicker.C:
			if err := sender.performDeliveryChecks(); err != nil {
				sender.log.Error().
					Error(err).
					Msg("failed to perform delivery check")
			}
		}
	}
}

func (sender *Sender) performDeliveryChecks() error {
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

	// TODO: group check datas by url

	performAgainChecksData := make([]deliveryCheckData, 0)
	counter := deliveryTypesCounter{}

	for i := range checksData {
		checksData[i].AttemptsCount += 1
		rspCode, rspBody, err := sender.doCheckRequest(checksData[i])
		if err != nil {
			sender.log.Warning().
				Error(err).
				String("url", checksData[i].URL).
				Int("delivery.check.response.code", rspCode).
				Msg("check request failed")

			// TODO: new state moira.DeliveryStateException, need to handle it
			continue
		}

		if !isAllowedResponseCode(rspCode) {
			sender.log.Warning().
				Int("delivery.check.response.code", rspCode).
				Interface("delivery.check.response.body", rspBody).
				Msg("not allowed response code")

			// TODO: new state moira.DeliveryStateException, need to handle it
			continue
		}

		populater := templating.NewWebhookDeliveryCheckPopulater(
			&templating.Contact{
				Type:  checksData[i].Contact.Type,
				Value: checksData[i].Contact.Value,
			},
			rspBody)

		resState, err := populater.Populate(sender.deliveryConfig.CheckTemplate)
		if err != nil {
			resState = moira.DeliveryStateUserException
			sender.log.Error().
				Error(err).
				Msg("error while populating check template")
		}

		newCheckData, scheduleAgain := handleStateTransition(checksData[i], resState, sender.deliveryConfig.MaxAttemptsCount, &counter)
		if scheduleAgain {
			performAgainChecksData = append(performAgainChecksData, newCheckData)
		}
	}

	// TODO: store checks data that needs to be checked again
	sender.storeChecksDataToCheckAgain(performAgainChecksData)
	// TODO: clean outdated check infos
	sender.removeOutdatedDeliveryChecks(fetchTimestamp)

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

func (sender *Sender) storeChecksDataToCheckAgain(checksData []deliveryCheckData) {
	if len(checksData) == 0 {
		return
	}

	scheduleAtTimestamp := sender.clock.NowUnix() + int64(sender.deliveryConfig.ReschedulingDelay)

	for _, data := range checksData {
		encoded, err := json.Marshal(data)
		if err != nil {
			sender.log.Warning().
				Error(err).
				Msg("failed to marshal data to check again")
			continue
		}

		// TODO: retry operations
		err = sender.Database.AddDeliveryChecksData(sender.contactType, scheduleAtTimestamp, string(encoded))
		if err != nil {
			sender.log.Error().
				String("check.url", data.URL).
				Error(err).
				Msg("failed to store check data")
			continue
		}
	}
}

func (sender *Sender) removeOutdatedDeliveryChecks(lastFetchTimestamp int64) error {
	_, err := sender.Database.RemoveDeliveryChecksData(sender.contactType, "-inf", strconv.FormatInt(lastFetchTimestamp, 1))
	return err
}

func (sender *Sender) scheduleDeliveryCheck(atTimestamp int64, sendAlertResponseBody []byte, contact moira.ContactData) error {
	var rspData map[string]interface{}
	err := json.Unmarshal(sendAlertResponseBody, &rspData)
	if err != nil {
		return fmt.Errorf("failed to unmarshal send alert response into json: %w", err)
	}

	urlPopulater := templating.NewWebhookDeliveryCheckURLPopulater(
		&templating.Contact{
			Type:  contact.Type,
			Value: contact.Value,
		},
		rspData)

	requestURL, err := urlPopulater.Populate(sender.deliveryConfig.URLTemplate)
	if err != nil {
		return fmt.Errorf("failed to fill url template with data: %w", err)
	}

	if err = validateURL(requestURL); err != nil {
		return fmt.Errorf("got bad url for check request: %w", err)
	}

	checkData := deliveryCheckData{
		URL:           requestURL,
		PreviousState: moira.DeliveryStatePending,
		Contact:       contact,
		AttemptsCount: 0,
	}

	encodedCheckData, err := json.Marshal(checkData)
	if err != nil {
		return fmt.Errorf("failed to encode check data: %w", err)
	}

	err = sender.Database.AddDeliveryChecksData(sender.contactType, atTimestamp, string(encodedCheckData))
	if err != nil {
		return fmt.Errorf("failed to save check data: %w", err)
	}

	return nil
}

func validateURL(requestURL string) error {
	urlStruct, err := url.Parse(requestURL)
	if err != nil {
		return err
	}

	if !(urlStruct.Scheme == "http://" || urlStruct.Scheme == "https://") {
		return fmt.Errorf("bad url scheme: %s", urlStruct.Scheme)
	}

	if urlStruct.Host == "" {
		return fmt.Errorf("host is empty")
	}

	return nil
}

type deliveryTypesCounter struct {
	deliveryOK      int64
	deliveryFailed  int64
	deliveryStopped int64
}

func handleStateTransition(checkData deliveryCheckData, newState string, maxAttemptsCount uint64, counter *deliveryTypesCounter) (deliveryCheckData, bool) {
	switch newState {
	case moira.DeliveryStateOK:
		counter.deliveryOK += 1
		return deliveryCheckData{}, false
	case moira.DeliveryStatePending, moira.DeliveryStateException:
		if checkData.AttemptsCount < maxAttemptsCount {
			checkData.PreviousState = newState
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
		// TODO: log unknown result of filing check template
		return deliveryCheckData{}, false
	}
}

func markMetrics(senderMetrics *metrics.SenderMetrics, counter *deliveryTypesCounter) {
	senderMetrics.ContactDeliveryNotificationOK.Mark(counter.deliveryOK)
	senderMetrics.ContactDeliveryNotificationFailed.Mark(counter.deliveryFailed)
	senderMetrics.ContactDeliveryNotificationCheckStopped.Mark(counter.deliveryStopped)
}
