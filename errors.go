package moira

//SenderBrokenContactError means than sender has no way to send message to contact.
// Maybe receive contact was deleted, blocked or archived.
type SenderBrokenContactError struct {
	SenderError error
}

func NewSenderBrokenContactError(senderError error) SenderBrokenContactError {
	return SenderBrokenContactError{
		SenderError: senderError,
	}
}

func (e SenderBrokenContactError) Error() string {
	return e.SenderError.Error()
}
