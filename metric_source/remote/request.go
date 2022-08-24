package remote

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

func (remote *Remote) prepareRequest(from, until int64, target string) (*http.Request, error) {
	req, err := http.NewRequest("GET", remote.config.URL, nil)
	if err != nil {
		return nil, err
	}
	q := req.URL.Query()
	q.Add("format", "json")
	q.Add("from", strconv.FormatInt(from, 10))
	q.Add("target", target)
	q.Add("until", strconv.FormatInt(until, 10))
	req.URL.RawQuery = q.Encode()
	if remote.config.User != "" && remote.config.Password != "" {
		req.SetBasicAuth(remote.config.User, remote.config.Password)
	}
	return req, nil
}

func (remote *Remote) makeRequest(req *http.Request) (body []byte, isRemoteAvailable bool, err error) {
	resp, err := remote.client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return body, false, fmt.Errorf(
			"the remote server is not available or the response was reset by timeout. Url: %s, Error: %v ",
			req.URL.String(),
			err,
		)
	}

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return body, false, err
	}

	if isRemoteUnavailableStatusCode(resp.StatusCode) {
		return body, false, fmt.Errorf(
			"the remote server is not available. Response status %d: %s", resp.StatusCode, string(body),
		)
	} else if resp.StatusCode != http.StatusOK {
		return body, true, fmt.Errorf("remote server response status %d: %s", resp.StatusCode, string(body))
	}

	return body, true, nil
}

func isRemoteUnavailableStatusCode(statusCode int) bool {
	switch statusCode {
	case http.StatusUnauthorized,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func (remote *Remote) makeRequestWithRetries(
	req *http.Request,
	requestTimeout time.Duration,
	retrySeconds []time.Duration,
) (body []byte, isRemoteAvailable bool, err error) {
	for attemptIndex := 0; attemptIndex < len(retrySeconds)+1; attemptIndex++ {
		body, isRemoteAvailable, err = remote.makeRequestWithTimeout(req, requestTimeout)
		if err == nil || isRemoteAvailable {
			return body, true, err
		}
		if attemptIndex < len(retrySeconds) {
			remote.clock.Sleep(retrySeconds[attemptIndex])
		}
	}
	return nil, false, err
}

func (remote *Remote) makeRequestWithTimeout(
	req *http.Request,
	requestTimeout time.Duration,
) (body []byte, isRemoteAvailable bool, err error) {
	if requestTimeout > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
		defer cancel()
		req = req.WithContext(ctx)
	}
	return remote.makeRequest(req)
}
