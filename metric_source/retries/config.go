package retries

import "time"

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
