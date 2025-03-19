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

// CheckNotificationsDelivery performs delivery checks for data fetched from db.
// Returns check data, that need to be scheduled again and moira.DeliveryTypesCounter,
// that contains the results of performing delivery checks for this iteration.
func (sender *Sender) CheckNotificationsDelivery(fetchedData []string) ([]string, moira.DeliveryTypesCounter) {
	counter := moira.DeliveryTypesCounter{}

	checksData := sender.unmarshalChecksData(fetchedData)
	if len(checksData) == 0 {
		return nil, counter
	}

	checksData = removeDuplicatedChecksData(checksData)

	checkAgainChecksData := make([]deliveryCheckData, 0)

	for i := range checksData {
		prevDeliveryStopped := counter.DeliveryChecksStopped
		extendedLogger := addContactFieldsToLog(sender.log.Clone(), checksData[i].Contact).
			String(moira.LogFieldNameTriggerID, checksData[i].TriggerID).
			String(logFieldNameDeliveryCheckUrl, checksData[i].URL)

		var deliveryState string
		checksData[i], deliveryState = sender.performSingleDeliveryCheck(&extendedLogger, checksData[i])

		newCheckData, scheduleAgain := handleStateTransition(checksData[i], deliveryState, sender.deliveryCheckConfig.MaxAttemptsCount, &counter)
		if scheduleAgain {
			checkAgainChecksData = append(checkAgainChecksData, newCheckData)
		}

		if !scheduleAgain {
			eventBuilder := extendedLogger.Warning()
			if prevDeliveryStopped != counter.DeliveryChecksStopped {
				eventBuilder = extendedLogger.Error()
			}

			eventBuilder.String(delivery.LogFieldPrefix+"state", deliveryState).
				Msg("Stop delivery checks")
		}
	}

	marshaledChecks := make([]string, 0, len(checkAgainChecksData))
	for _, data := range checkAgainChecksData {
		marshaledData, err := json.Marshal(data)
		if err != nil {
			addContactFieldsToEventBuilder(sender.log.Warning(), data.Contact).
				String(logFieldNameDeliveryCheckUrl, data.URL).
				String(moira.LogFieldNameTriggerID, data.TriggerID).
				Msg("Failed to marshal delivery check to check again")
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
				Msg("Failed to unmarshal encoded data")
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

func (sender *Sender) performSingleDeliveryCheck(extendedLogger *moira.Logger, checkData deliveryCheckData) (deliveryCheckData, string) {
	var deliveryState string
	checkData.AttemptsCount += 1

	rspCode, rspBody, err := sender.doDeliveryCheckRequest(checkData)
	*extendedLogger = addDeliveryCheckFieldsToLog(*extendedLogger, rspCode, string(rspBody))
	if err != nil {
		(*extendedLogger).Error().
			Error(err).
			Msg("Check request failed")

		return checkData, moira.DeliveryStateException
	}

	if !isAllowedResponseCode(rspCode) {
		(*extendedLogger).Error().
			Msg("Not allowed response code")

		return checkData, moira.DeliveryStateException
	}

	var unmarshalledBody map[string]interface{}
	err = json.Unmarshal(rspBody, &unmarshalledBody)
	if err != nil {
		(*extendedLogger).Error().
			Error(err).
			Msg("Failed to unmarshal response")

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
		(*extendedLogger).Error().
			Error(err).
			Msg("Error while populating check template")

		return checkData, moira.DeliveryStateUserException
	}

	if _, ok := moira.DeliveryStatesSet[deliveryState]; !ok {
		(*extendedLogger).Error().
			Msg("check template returned unknown delivery state")
	}

	return checkData, deliveryState
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

func addDeliveryCheckFieldsToLog(logger moira.Logger, rspCode int, body string) moira.Logger {
	return logger.
		Int(logFieldNameDeliveryCheckResponseCode, rspCode).
		String(logFieldNameDeliveryCheckResponseBody, body)
}

func addContactFieldsToLog(logger moira.Logger, contact moira.ContactData) moira.Logger {
	return logger.
		String(moira.LogFieldNameContactID, contact.ID).
		String(moira.LogFieldNameContactType, contact.Type).
		String(moira.LogFieldNameContactValue, contact.Value)
}

func addContactFieldsToEventBuilder(eventBuilder logging.EventBuilder, contact moira.ContactData) logging.EventBuilder {
	return eventBuilder.
		String(moira.LogFieldNameContactID, contact.ID).
		String(moira.LogFieldNameContactType, contact.Type).
		String(moira.LogFieldNameContactValue, contact.Value)
}
