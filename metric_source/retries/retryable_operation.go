package retries

// RetryableOperation is an action that can be retried after some time interval.
// If there is an error than should not be retried wrap it into backoff.PermanentError.
type RetryableOperation[T any] interface {
	DoRetryableOperation() (T, error)
}
