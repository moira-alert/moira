package retries

import (
	"github.com/cenkalti/backoff/v4"
)

// Retrier retries the given operation with given backoff.
type Retrier[T any] interface {
	// Retry the given operation until the op succeeds or op returns backoff.PermanentError or backoffPolicy returns backoff.Stop.
	Retry(op RetryableOperation[T], backoffPolicy backoff.BackOff) (T, error)
}

type standardRetrier[T any] struct{}

// NewStandardRetrier returns standard retrier which will perform retries
// according to backoff policy provided by BackoffFactory.
func NewStandardRetrier[T any]() Retrier[T] {
	return standardRetrier[T]{}
}

// Retry the given operation until the op succeeds or op returns backoff.PermanentError or backoffPolicy returns backoff.Stop.
func (r standardRetrier[T]) Retry(op RetryableOperation[T], backoffPolicy backoff.BackOff) (T, error) {
	return backoff.RetryWithData[T](op.DoRetryableOperation, backoffPolicy)
}
