package retries

import (
	"errors"
	"time"
)

// Config for exponential backoff retries.
type Config struct {
	// InitialInterval between requests.
	InitialInterval time.Duration
	// RandomizationFactor is used in exponential backoff to add some randomization
	// when calculating next interval between requests.
	// It will be used in multiplication like:
	//	RandomizedInterval = RetryInterval * (random value in range [1 - RandomizationFactor, 1 + RandomizationFactor])
	RandomizationFactor float64
	// Each new RetryInterval will be multiplied on Multiplier.
	Multiplier float64
	// MaxInterval is the cap for RetryInterval. Note that it doesn't cap the RandomizedInterval.
	MaxInterval time.Duration
	// MaxElapsedTime caps the time passed from first try. If time passed is greater than MaxElapsedTime than stop retrying.
	MaxElapsedTime time.Duration
	// MaxRetriesCount is the amount of allowed retries. So at most MaxRetriesCount will be performed.
	MaxRetriesCount uint64
}

var (
	errNoInitialInterval                  = errors.New("initial_interval must be specified and can't be 0")
	errNoMaxInterval                      = errors.New("max_interval must be specified and can't be 0")
	errNoMaxElapsedTimeAndMaxRetriesCount = errors.New("at least one of max_elapsed_time, max_retries_count must be specified")
)

// Validate checks that retries Config has all necessary fields.
func (conf Config) Validate() error {
	resErrors := make([]error, 0)

	if conf.InitialInterval == 0 {
		resErrors = append(resErrors, errNoInitialInterval)
	}

	if conf.MaxInterval == 0 {
		resErrors = append(resErrors, errNoMaxInterval)
	}

	if conf.MaxElapsedTime == 0 && conf.MaxRetriesCount == 0 {
		resErrors = append(resErrors, errNoMaxElapsedTimeAndMaxRetriesCount)
	}

	return errors.Join(resErrors...)
}
