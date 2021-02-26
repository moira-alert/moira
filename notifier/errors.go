package notifier

//SenderBrokenContactError means than sender has no way to send message to contact.
// Maybe receive contact was deleted, blocked or archived.
type SenderBrokenContactError struct {
	SenderError error
}

func (e *SenderBrokenContactError) Error() string {
	return e.Error()
}
