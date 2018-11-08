package notifications

// notifierInBadStateError is used for ERROR state of notifier service
type notifierInBadStateError struct {
	message string
}

// notifierInBadStateError implementation with constant error message
func (err notifierInBadStateError) Error() string {
	return err.message
}
