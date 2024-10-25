package retries

import (
	"time"
)

// Config for exponential backoff retries.
type Config struct {
	// InitialInterval between requests.
	InitialInterval time.Duration `validate:"required,gt=0s"`
	// RandomizationFactor is used in exponential backoff to add some randomization
	// when calculating next interval between requests.
	// It will be used in multiplication like:
	//	RandomizedInterval = RetryInterval * (random value in range [1 - RandomizationFactor, 1 + RandomizationFactor])
	RandomizationFactor float64
	// Each new RetryInterval will be multiplied on Multiplier.
	Multiplier float64
	// MaxInterval is the cap for RetryInterval. Note that it doesn't cap the RandomizedInterval.
	MaxInterval time.Duration `validate:"required,gt=0s"`
	// MaxElapsedTime caps the time passed from first try. If time passed is greater than MaxElapsedTime than stop retrying.
	MaxElapsedTime time.Duration `validate:"required_if=MaxRetriesCount 0"`
	// MaxRetriesCount is the amount of allowed retries. So at most MaxRetriesCount will be performed.
	MaxRetriesCount uint64 `validate:"required_if=MaxElapsedTime 0"`
}
