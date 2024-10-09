package remote

import (
	"fmt"
	"github.com/moira-alert/moira/metric_source/retries"
	"net/http"
	"time"

	"github.com/moira-alert/moira/clock"

	"github.com/moira-alert/moira"
	metricSource "github.com/moira-alert/moira/metric_source"
)

// ErrRemoteStorageDisabled is used to prevent remote.Fetch calls when remote storage is disabled.
var ErrRemoteStorageDisabled = fmt.Errorf("remote graphite storage is not enabled")

// ErrRemoteTriggerResponse is a custom error when remote trigger check fails.
type ErrRemoteTriggerResponse struct {
	InternalError error
	Target        string
}

// Error is a representation of Error interface method.
func (err ErrRemoteTriggerResponse) Error() string {
	return err.InternalError.Error()
}

// ErrRemoteUnavailable is a custom error when remote trigger check fails.
type ErrRemoteUnavailable struct {
	InternalError error
	Target        string
}

// Error is a representation of Error interface method.
func (err ErrRemoteUnavailable) Error() string {
	return err.InternalError.Error()
}

// Remote is implementation of MetricSource interface, which implements fetch metrics method from remote graphite installation.
type Remote struct {
	config                    *Config
	client                    *http.Client
	clock                     moira.Clock
	retrier                   retries.Retrier[[]byte]
	requestBackoffFactory     retries.BackoffFactory
	healthcheckBackoffFactory retries.BackoffFactory
}

// Create configures remote metric source.
func Create(config *Config) (metricSource.MetricSource, error) {
	if config.URL == "" {
		return nil, fmt.Errorf("remote graphite URL should not be empty")
	}

	var (
		requestBackoffFactory     retries.BackoffFactory
		healthcheckBackoffFactory retries.BackoffFactory
	)

	requestBackoffFactory = retries.NewExponentialBackoffFactory(config.Retries)
	if config.HealthcheckRetries != nil {
		healthcheckBackoffFactory = retries.NewExponentialBackoffFactory(*config.HealthcheckRetries)
	} else {
		healthcheckBackoffFactory = requestBackoffFactory
	}

	return &Remote{
		config:                    config,
		client:                    &http.Client{Timeout: config.Timeout},
		clock:                     clock.NewSystemClock(),
		retrier:                   retries.NewStandardRetrier[[]byte](),
		requestBackoffFactory:     requestBackoffFactory,
		healthcheckBackoffFactory: healthcheckBackoffFactory,
	}, nil
}

// Fetch fetches remote metrics and converts them to expected format.
func (remote *Remote) Fetch(target string, from, until int64, allowRealTimeAlerting bool) (metricSource.FetchResult, error) {
	// Don't fetch intervals larger than metrics TTL to prevent OOM errors
	// See https://github.com/moira-alert/moira/pull/519
	from = moira.MaxInt64(from, until-int64(remote.config.MetricsTTL.Seconds()))

	req, err := remote.prepareRequest(from, until, target)
	if err != nil {
		return nil, ErrRemoteTriggerResponse{
			InternalError: err,
			Target:        target,
		}
	}

	body, err := remote.makeRequest(req)
	if err != nil {
		return nil, internalErrToPublicErr(err, target)
	}

	resp, err := decodeBody(body)
	if err != nil {
		return nil, ErrRemoteTriggerResponse{
			InternalError: err,
			Target:        target,
		}
	}

	fetchResult := convertResponse(resp, allowRealTimeAlerting)
	return &fetchResult, nil
}

// GetMetricsTTLSeconds returns maximum time interval that we are allowed to fetch from remote.
func (remote *Remote) GetMetricsTTLSeconds() int64 {
	return int64(remote.config.MetricsTTL.Seconds())
}

// IsAvailable checks if graphite API is available and returns 200 response.
func (remote *Remote) IsAvailable() (bool, error) {
	until := time.Now().Unix()
	from := until - 600 //nolint

	req, err := remote.prepareRequest(from, until, "NonExistingTarget")
	if err != nil {
		return false, err
	}

	_, err = remote.makeRequest(req)
	err = internalErrToPublicErr(err, "")

	return !isRemoteUnavailable(err), err
}
