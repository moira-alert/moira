package webhook

import (
	"fmt"
	"net/http"
)

func (sender *Sender) buildDeliveryCheckRequest(checkData deliveryCheckData) (*http.Request, error) {
	return buildRequest(sender.log, http.MethodGet, checkData.URL, nil, sender.deliveryCheckConfig.User, sender.deliveryCheckConfig.Password, sender.deliveryCheckConfig.Headers)
}

func (sender *Sender) doDeliveryCheckRequest(checkData deliveryCheckData) (int, []byte, error) {
	req, err := sender.buildDeliveryCheckRequest(checkData)
	if err != nil {
		return 0, nil, err
	}

	statusCode, body, err := performRequest(sender.client, req)
	if err != nil {
		return 0, nil, fmt.Errorf("check delivery request failed: %w", err)
	}

	return statusCode, body, nil
}
