package webhook

import (
	"encoding/json"
	"fmt"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/logging"
	"github.com/moira-alert/moira/senders/delivery"
	"github.com/moira-alert/moira/templating"
)

const (
	logFieldNameDeliveryCheckUrl             = delivery.LogFieldPrefix + "url"
	logFieldNameDeliveryCheckResponseCode    = delivery.LogFieldPrefix + "response.code"
	logFieldNameDeliveryCheckResponseBody    = delivery.LogFieldPrefix + "response.body"
	logFieldNameDeliveryCheckUnknownState    = delivery.LogFieldPrefix + "unknown.state"
	logFieldNameSendNotificationResponseBody = delivery.LogFieldPrefix + "response.body"
)

type deliveryCheckData struct {
	// URL for delivery check request.
	URL string `json:"url"`
	// Contact related to delivery check.
	Contact moira.ContactData `json:"contact"`
	// TriggerID for which notification was generated.
	TriggerID string `json:"trigger_id"`
	// AttemptsCount of performing delivery check.
	AttemptsCount uint64 `json:"attempts_count"`
}

func (sender *Sender) CheckNotificationsDelivery(fetchedDeliveryChecks []string) ([]string, moira.DeliveryTypesCounter) {
	counter := moira.DeliveryTypesCounter{}

	checksData := sender.unmarshalChecksData(fetchedDeliveryChecks)
	if len(checksData) == 0 {
		return nil, counter
	}

	checksData = removeDuplicatedChecksData(checksData)

	checkAgainChecksData := make([]deliveryCheckData, 0)

	for i := range checksData {
		prevDeliveryStopped := counter.DeliveryChecksStopped

		var deliveryState string
		checksData[i], deliveryState = sender.performSingleDeliveryCheck(checksData[i])

		newCheckData, scheduleAgain := handleStateTransition(checksData[i], deliveryState, sender.deliveryCheckConfig.MaxAttemptsCount, &counter)
		if scheduleAgain {
			checkAgainChecksData = append(checkAgainChecksData, newCheckData)
		}

		if prevDeliveryStopped != counter.DeliveryChecksStopped || deliveryState == moira.DeliveryStateFailed {
			addContactFieldsToLog(sender.log.Error(), checksData[i].Contact).
				String(logFieldNameDeliveryCheckUrl, checksData[i].URL).
				String(moira.LogFieldNameTriggerID, checksData[i].TriggerID).
				String(delivery.LogFieldPrefix+"state", deliveryState).
				Msg("stop delivery checks")
		}
	}

	marshaledChecks := make([]string, 0, len(checkAgainChecksData))
	for _, data := range checkAgainChecksData {
		marshaledData, err := json.Marshal(data)
		if err != nil {
			addContactFieldsToLog(sender.log.Warning(), data.Contact).
				String(logFieldNameDeliveryCheckUrl, data.URL).
				String(moira.LogFieldNameTriggerID, data.TriggerID).
				Msg("failed to marshal delivery check to check again")
			continue
		}

		marshaledChecks = append(marshaledChecks, string(marshaledData))
	}

	return marshaledChecks, counter
}

func (sender *Sender) unmarshalChecksData(marshaledData []string) []deliveryCheckData {
	checksData := make([]deliveryCheckData, 0, len(marshaledData))

	for _, encoded := range marshaledData {
		var data deliveryCheckData
		err := json.Unmarshal([]byte(encoded), &data)
		if err != nil {
			sender.log.Warning().
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
	deduplicated := make([]deliveryCheckData, 0)
	uniqueMap := make(map[string]int, len(checksData))

	for _, checkData := range checksData {
		key := fmt.Sprintf("%s_%s_%s_%s", checkData.URL, checkData.Contact.ID, checkData.Contact.Value, checkData.TriggerID)
		if prevCheckDataIndex, ok := uniqueMap[key]; ok {
			if deduplicated[prevCheckDataIndex].AttemptsCount < checkData.AttemptsCount {
				deduplicated[prevCheckDataIndex] = checkData
			}
		} else {
			uniqueMap[key] = len(deduplicated)
			deduplicated = append(deduplicated, checkData)
		}
	}

	return deduplicated
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

func handleStateTransition(checkData deliveryCheckData, newState string, maxAttemptsCount uint64, counter *moira.DeliveryTypesCounter) (deliveryCheckData, bool) {
	switch newState {
	case moira.DeliveryStateOK:
		counter.DeliveryOK += 1
		return deliveryCheckData{}, false
	case moira.DeliveryStatePending, moira.DeliveryStateException:
		if checkData.AttemptsCount < maxAttemptsCount {
			return checkData, true
		}

		counter.DeliveryChecksStopped += 1
		return deliveryCheckData{}, false
	case moira.DeliveryStateFailed:
		counter.DeliveryFailed += 1
		return checkData, false
	case moira.DeliveryStateUserException:
		counter.DeliveryChecksStopped += 1
		return deliveryCheckData{}, false
	default:
		counter.DeliveryChecksStopped += 1
		return deliveryCheckData{}, false
	}
}
