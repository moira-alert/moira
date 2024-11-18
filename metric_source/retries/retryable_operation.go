package retries

// RetryableOperation is an action that can be retried after some time interval.
// If there is an error in DoRetryableOperation that should not be retried, wrap the error with backoff.PermanentError.
type RetryableOperation[T any] interface {
	DoRetryableOperation() (T, error)
}
