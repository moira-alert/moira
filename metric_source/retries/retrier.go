package retries

import (
	"github.com/cenkalti/backoff/v4"
)

// Retrier retries the given operation.
type Retrier[T any] interface {
	Retry(op RetryableOperation[T]) (T, error)
}

type standardRetrier[T any] struct {
	backoffFactory BackoffFactory
}

// NewStandardRetrier returns standard retrier which will perform retries
// according to backoff policy provided by BackoffFactory.
func NewStandardRetrier[T any](backoffFactory BackoffFactory) Retrier[T] {
	return standardRetrier[T]{
		backoffFactory: backoffFactory,
	}
}

func (r standardRetrier[T]) Retry(op RetryableOperation[T]) (T, error) {
	return backoff.RetryWithData[T](op.DoRetryableOperation, r.backoffFactory.NewBackOff())
}
