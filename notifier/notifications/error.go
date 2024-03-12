package notifications

// notifierInBadStateError is used for ERROR state of notifier service.
type notifierInBadStateError string

// notifierInBadStateError implementation with constant error message.
func (err notifierInBadStateError) Error() string {
	return string(err)
}
