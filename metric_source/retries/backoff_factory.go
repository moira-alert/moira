package retries

import "github.com/cenkalti/backoff/v4"

type BackoffFactory interface {
	NewBackOff() backoff.BackOff
}

type ExponentialBackoffFactory struct {
	config Config
}

func NewExponentialBackoffFactory(config Config) BackoffFactory {
	return ExponentialBackoffFactory{
		config: config,
	}
}

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
