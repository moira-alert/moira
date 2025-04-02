package webhook

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/moira-alert/moira"
)

func buildRequest(
	logger moira.Logger,
	method string,
	requestURL string,
	body []byte,
	user string,
	password string,
	headers map[string]string,
) (*http.Request, error) {
	request, err := http.NewRequestWithContext(context.TODO(), method, requestURL, bytes.NewBuffer(body))
	if err != nil {
		return request, err
	}

	if user != "" && password != "" {
		request.SetBasicAuth(user, password)
	}

	for k, v := range headers {
		request.Header.Set(k, v)
	}

	logger.Debug().
		String("method", request.Method).
		String("url", request.URL.String()).
		String("body", string(body)).
		Msg("Created request")

	return request, nil
}

func performRequest(client *http.Client, request *http.Request) (int, []byte, error) {
	rsp, err := client.Do(request)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to perform request: %w", err)
	}
	defer rsp.Body.Close()

	bodyBytes, err := io.ReadAll(rsp.Body)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return rsp.StatusCode, bodyBytes, nil
}

func isAllowedResponseCode(responseCode int) bool {
	return (responseCode >= http.StatusOK) && (responseCode < http.StatusMultipleChoices)
}
