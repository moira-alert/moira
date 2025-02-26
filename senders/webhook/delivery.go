package webhook

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database"
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
	Trigger       moira.TriggerData `json:"trigger"`
	AttemptsCount uint              `json:"attempts_count"`
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
	checkTicker := time.NewTicker(time.Duration(sender.deliveryCfg.CheckTimeout) * time.Second)

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
	now := sender.clock.NowUnix()

	marshaledData, err := sender.Database.GetDeliveryChecksData(sender.contactType, "-inf", strconv.FormatInt(now, 10))
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
	var (
		deliverOK     int64
		deliverFailed int64
	)

	for i := range checksData {
		checksData[i].AttemptsCount += 1
		rspCode, rspBody, err := sender.doCheckRequest(checksData[i])
		if err != nil {
			sender.log.Warning().
				Error(err).
				String("url", checksData[i].URL).
				Int("delivery.check.response.code", rspCode).
				Msg("check request failed")
			checksData[i].PreviousState = moira.DeliveryStateException
			continue
		}

		if !isAllowedResponseCode(rspCode) {
			sender.log.Warning().
				Int("delivery.check.response.code", rspCode).
				Interface("delivery.check.response.body", rspBody).
				Msg("not allowed response code")
			checksData[i].PreviousState = moira.DeliveryStateException
			continue
		}

		populater := templating.NewWebhookDeliveryCheckPopulater(
			&templating.Contact{
				Type:  checksData[i].Contact.Type,
				Value: checksData[i].Contact.Value,
			},
			rspBody)

		resState, err := populater.Populate(sender.deliveryCfg.CheckTemplate)
		if err != nil {
			resState = moira.DeliveryStateUserException
			sender.log.Error().
				Error(err).
				Msg("error while populating check template")
		}

		switch resState {
		case moira.DeliveryStateOK:
			deliverOK += 1
		case moira.DeliveryStatePending, moira.DeliveryStateException:
			// TODO: use checksData[i].AttemptsCount
			checksData[i].PreviousState = resState
			performAgainChecksData = append(performAgainChecksData, checksData[i])
		case moira.DeliveryStateFailed:
			deliverFailed += 1
		case moira.DeliveryStateUserException:
			// TODO: what do here?
		default:
			// TODO: can be same as Pending or UserException ?
		}
	}

	// TODO: store checks data that needs to be checked again
	// TODO: clean outdated check infos

	sender.metrics.ContactDeliveryNotificationOK.Mark(deliverOK)
	sender.metrics.ContactDeliveryNotificationFailed.Mark(deliverFailed)

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

func (sender *Sender) doCheckRequest(checkData deliveryCheckData) (int, map[string]interface{}, error) {
	req, err := http.NewRequest(http.MethodGet, checkData.URL, nil)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to create new request: %w", err)
	}

	if sender.deliveryCfg.User != "" && sender.deliveryCfg.Password != "" {
		req.SetBasicAuth(sender.user, sender.password)
	}

	for k, v := range sender.deliveryCfg.Headers {
		req.Header.Set(k, v)
	}

	rsp, err := sender.client.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to do request: %w", err)
	}
	defer func() { _ = rsp.Body.Close() }()

	bodyBytes, err := io.ReadAll(rsp.Body)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var rspMap map[string]interface{}
	err = json.Unmarshal(bodyBytes, &rspMap)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to unmarshal body into json: %w", err)
	}

	return rsp.StatusCode, rspMap, nil
}

func (sender *Sender) storeChecksDataToCheckAgain(checksData []deliveryCheckData) error {
	if len(checksData) == 0 {
		return nil
	}

	scheduleAtTimestamp := sender.clock.NowUnix() + int64(sender.deliveryCfg.ReschedulingDelay)

	for _, data := range checksData {
		encoded, err := json.Marshal(data)
		if err != nil {
			sender.log.Warning().
				Error(err).
				Msg("failed to marshal data to check again")
			continue
		}

		// TODO: retry operations
		err := sender.Database.AddDeliveryChecksData(sender.contactType, scheduleAtTimestamp, string(encoded))
	}
}
