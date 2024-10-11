package remote

import (
	"errors"
	"github.com/moira-alert/moira/metric_source/retries"
	"time"
)

// Config represents config from remote storage.
type Config struct {
	URL                string
	CheckInterval      time.Duration
	MetricsTTL         time.Duration
	Timeout            time.Duration
	User               string
	Password           string
	HealthcheckTimeout time.Duration
	Retries            retries.Config
	HealthcheckRetries retries.Config
}

var (
	errBadRemoteUrl         = errors.New("remote graphite URL should not be empty")
	errNoTimeout            = errors.New("timeout must be specified and can't be 0")
	errNoHealthcheckTimeout = errors.New("healthcheck_timeout must be specified and can't be 0")
)

func (conf Config) validate() error {
	resErrors := make([]error, 0)

	if conf.URL == "" {
		resErrors = append(resErrors, errBadRemoteUrl)
	}

	if conf.Timeout == 0 {
		resErrors = append(resErrors, errNoTimeout)
	}

	if conf.HealthcheckTimeout == 0 {
		resErrors = append(resErrors, errNoHealthcheckTimeout)
	}

	if errRetriesValidate := conf.Retries.Validate(); errRetriesValidate != nil {
		resErrors = append(resErrors, errRetriesValidate)
	}

	if errHealthcheckRetriesValidate := conf.HealthcheckRetries.Validate(); errHealthcheckRetriesValidate != nil {
		resErrors = append(resErrors, errHealthcheckRetriesValidate)
	}

	return errors.Join(resErrors...)
}
