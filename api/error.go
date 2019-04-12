package api

// ErrInvalidRequestContent used as custom error for dto logical validations
type ErrInvalidRequestContent struct {
	ValidationError error
}

// Error is a representation of Error interface method
func (err ErrInvalidRequestContent) Error() string {
	return err.ValidationError.Error()
}
