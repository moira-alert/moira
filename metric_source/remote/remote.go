package remote

import (
	"net/http"
	"time"

	"github.com/moira-alert/moira/metric_source/retries"

	"github.com/moira-alert/moira"
	metricSource "github.com/moira-alert/moira/metric_source"
)

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
	retrier                   retries.Retrier[[]byte]
	requestBackoffFactory     retries.BackoffFactory
	healthcheckBackoffFactory retries.BackoffFactory
}

// Create configures remote metric source.
func Create(config *Config) (metricSource.MetricSource, error) {
	if err := config.validate(); err != nil {
		return nil, err
	}

	return &Remote{
		config:                    config,
		client:                    &http.Client{},
		retrier:                   retries.NewStandardRetrier[[]byte](),
		requestBackoffFactory:     retries.NewExponentialBackoffFactory(config.Retries),
		healthcheckBackoffFactory: retries.NewExponentialBackoffFactory(config.HealthcheckRetries),
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

	body, err := remote.makeRequest(req, remote.config.Timeout, remote.requestBackoffFactory.NewBackOff())
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

	_, err = remote.makeRequest(req, remote.config.HealthcheckTimeout, remote.healthcheckBackoffFactory.NewBackOff())
	publicErr := internalErrToPublicErr(err, "")

	return !isRemoteUnavailable(publicErr), publicErr
}
