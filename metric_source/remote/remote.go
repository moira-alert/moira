package remote

import (
	"fmt"
	"time"

	"github.com/moira-alert/moira/metric_source"
)

// ErrRemoteStorageDisabled is used to prevent remote.Fetch calls when remote storage is disabled
var ErrRemoteStorageDisabled = fmt.Errorf("remote graphite storage is not enabled")

// ErrRemoteTriggerResponse is a custom error when remote trigger check fails
type ErrRemoteTriggerResponse struct {
	InternalError error
	Target        string
}

// Error is a representation of Error interface method
func (err ErrRemoteTriggerResponse) Error() string {
	return fmt.Sprintf("failed to get remote target '%s': %s", err.Target, err.InternalError.Error())
}

// Remote is implementation of MetricSource interface, which implements fetch metrics method from remote graphite installation
type Remote struct {
	config *Config
}

// CreateRemoteSource configures remote metric source
func CreateRemoteSource(config *Config) metricSource.MetricSource {
	return &Remote{
		config: config,
	}
}

// Fetch fetches remote metrics and converts them to expected format
func (remote *Remote) Fetch(target string, from, until int64, allowRealTimeAlerting bool) (metricSource.FetchResult, error) {
	req, err := prepareRequest(from, until, target, remote.config)
	if err != nil {
		return nil, ErrRemoteTriggerResponse{
			InternalError: err,
			Target:        target,
		}
	}
	body, err := makeRequest(req, remote.config.Timeout)
	if err != nil {
		return nil, ErrRemoteTriggerResponse{
			InternalError: err,
			Target:        target,
		}
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

// IsConfigured returns false in cases that user does not properly configure remote settings like graphite URL
func (remote *Remote) IsConfigured() (bool, error) {
	if remote.config.isEnabled() {
		return true, nil
	}
	return false, ErrRemoteStorageDisabled
}

// IsRemoteAvailable checks if graphite API is available and returns 200 response
func (remote *Remote) IsRemoteAvailable() (bool, error) {
	maxRetries := 3
	until := time.Now().Unix()
	from := until - 600
	req, err := prepareRequest(from, until, "NonExistingTarget", remote.config)
	if err != nil {
		return false, err
	}
	for attempt := 0; attempt < maxRetries; attempt++ {
		_, err = makeRequest(req, remote.config.Timeout)
		if err == nil {
			return true, nil
		}
	}
	return false, err
}
