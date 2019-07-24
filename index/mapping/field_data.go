package mapping

// FieldData is container for field-related parameters
type FieldData struct {
	name     string
	nameTag  string
	priority float64
}

// String returns TriggerField name.
func (field FieldData) String() string {
	return field.name
}

// GetTagValue returns TriggerField value used in marshalling.
func (field FieldData) GetTagValue() string {
	return field.nameTag
}

// GetPriority returns field priority
func (field FieldData) GetPriority() float64 {
	return field.priority
}
