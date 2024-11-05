package remote

import (
	"time"

	"github.com/moira-alert/moira/metric_source/retries"
)

// Config represents config from remote storage.
type Config struct {
	URL                string `validate:"required,url"`
	CheckInterval      time.Duration
	MetricsTTL         time.Duration
	Timeout            time.Duration `validate:"required,gt=0s"`
	User               string
	Password           string
	HealthcheckTimeout time.Duration `validate:"required,gt=0s"`
	Retries            retries.Config
	HealthcheckRetries retries.Config
}
