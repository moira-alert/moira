package retries

import "github.com/cenkalti/backoff/v4"

// BackoffFactory is used for creating backoff. It is expected that all backoffs created with one factory instance
// have the same behaviour.
type BackoffFactory interface {
	NewBackOff() backoff.BackOff
}

// ExponentialBackoffFactory is a factory that generates exponential backoffs based on given config.
type ExponentialBackoffFactory struct {
	config Config
}

// NewExponentialBackoffFactory creates new BackoffFactory which will generate exponential backoffs.
func NewExponentialBackoffFactory(config Config) BackoffFactory {
	return ExponentialBackoffFactory{
		config: config,
	}
}

// NewBackOff creates new backoff.
func (factory ExponentialBackoffFactory) NewBackOff() backoff.BackOff {
	backoffPolicy := backoff.NewExponentialBackOff(
		backoff.WithInitialInterval(factory.config.InitialInterval),
		backoff.WithRandomizationFactor(factory.config.RandomizationFactor),
		backoff.WithMultiplier(factory.config.Multiplier),
		backoff.WithMaxInterval(factory.config.MaxInterval),
		backoff.WithMaxElapsedTime(factory.config.MaxElapsedTime))

	if factory.config.MaxRetriesCount > 0 {
		return backoff.WithMaxRetries(backoffPolicy, factory.config.MaxRetriesCount)
	}

	return backoffPolicy
}
