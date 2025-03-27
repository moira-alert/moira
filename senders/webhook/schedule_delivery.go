package webhook

import (
	"encoding/json"
	"fmt"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/templating"
)

func (sender *Sender) scheduleDeliveryCheck(contact moira.ContactData, triggerID string, responseBody []byte) {
	extendedLogger := addContactFieldsToLog(sender.log.Clone(), contact).
		String(moira.LogFieldNameTriggerID, triggerID).
		String(logFieldNameSendNotificationResponseBody, string(responseBody))

	var responseData map[string]interface{}
	err := json.Unmarshal(responseBody, &responseData)
	if err != nil {
		extendedLogger.Error().
			Error(err).
			Msg("Failed to schedule delivery check because of not unmarshalling")
		return
	}

	checkData, err := prepareDeliveryCheck(contact, responseData, sender.deliveryCheckConfig.URLTemplate, triggerID)
	if err != nil {
		extendedLogger.Error().
			Error(err).
			Msg("Failed to prepare delivery check")
		return
	}

	extendedLogger = extendedLogger.String(logFieldNameDeliveryCheckUrl, checkData.URL)
	err = sender.addDeliveryChecks(checkData, sender.clock.NowUnix())
	if err != nil {
		extendedLogger.Error().
			Error(err).
			Msg("Failed to schedule delivery check")
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

	if err = moira.ValidateURL(requestURL); err != nil {
		return deliveryCheckData{}, fmt.Errorf("got bad url for check request: %w, url: %s", err, requestURL)
	}

	return deliveryCheckData{
		URL:           requestURL,
		Contact:       contact,
		TriggerID:     triggerID,
		AttemptsCount: 0,
	}, nil
}

func (sender *Sender) addDeliveryChecks(data deliveryCheckData, timestamp int64) error {
	encoded, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal check data: %w", err)
	}

	// TODO: retry operations?
	err = sender.Controller.AddDeliveryChecksData(timestamp, string(encoded))
	if err != nil {
		return fmt.Errorf("failed to store check data: %w", err)
	}

	return nil
}
