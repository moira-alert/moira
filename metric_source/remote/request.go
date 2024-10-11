package remote

import (
	"context"
	"errors"
	"fmt"
	"github.com/cenkalti/backoff/v4"
	"io"
	"net/http"
	"strconv"
	"time"
)

func (remote *Remote) prepareRequest(from, until int64, target string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, remote.config.URL, nil)
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

func (remote *Remote) makeRequest(req *http.Request, timeout time.Duration, backoffPolicy backoff.BackOff) ([]byte, error) {
	return remote.retrier.Retry(
		requestToRemoteGraphite{
			client:         remote.client,
			request:        req,
			requestTimeout: timeout,
		},
		backoffPolicy)
}

type requestToRemoteGraphite struct {
	client         *http.Client
	request        *http.Request
	requestTimeout time.Duration
}

func (r requestToRemoteGraphite) DoRetryableOperation() ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), r.requestTimeout)
	defer cancel()

	req := r.request.WithContext(ctx)

	resp, err := r.client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return nil, errRemoteUnavailable{
			internalErr: fmt.Errorf(
				"the remote server is not available or the response was reset by timeout. Url: %s, Error: %w ",
				req.URL.String(),
				err),
		}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return body, errRemoteUnavailable{internalErr: err}
	}

	if isRemoteUnavailableStatusCode(resp.StatusCode) {
		return body, errRemoteUnavailable{
			internalErr: fmt.Errorf(
				"the remote server is not available. Response status %d: %s",
				resp.StatusCode, string(body)),
		}
	} else if resp.StatusCode != http.StatusOK {
		return body, backoff.Permanent(
			errInvalidRequest{
				internalErr: fmt.Errorf("bad response status %d: %s", resp.StatusCode, string(body)),
			})
	}

	return body, nil
}

type errInvalidRequest struct {
	internalErr error
}

func (err errInvalidRequest) Error() string {
	return err.internalErr.Error()
}

type errRemoteUnavailable struct {
	internalErr error
}

func (err errRemoteUnavailable) Error() string {
	return err.internalErr.Error()
}

func isRemoteUnavailable(err error) bool {
	var errUnavailable ErrRemoteUnavailable
	return errors.As(err, &errUnavailable)
}

func internalErrToPublicErr(err error, target string) error {
	if err == nil {
		return nil
	}

	var invalidReqErr errInvalidRequest
	if errors.As(err, &invalidReqErr) {
		return ErrRemoteTriggerResponse{
			InternalError: invalidReqErr.internalErr,
			Target:        target,
		}
	}

	var errUnavailable errRemoteUnavailable
	if errors.As(err, &errUnavailable) {
		return ErrRemoteUnavailable{
			InternalError: errUnavailable.internalErr,
			Target:        target,
		}
	}

	return ErrRemoteTriggerResponse{}
}

var remoteUnavailableStatusCodes = map[int]struct{}{
	http.StatusUnauthorized:       {},
	http.StatusBadGateway:         {},
	http.StatusServiceUnavailable: {},
	http.StatusGatewayTimeout:     {},
}

func isRemoteUnavailableStatusCode(statusCode int) bool {
	_, isUnavailableCode := remoteUnavailableStatusCodes[statusCode]
	return isUnavailableCode
}
