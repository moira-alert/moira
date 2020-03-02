package mapping

// FieldData is container for field-related parameters
// name represents indexed object field name
// nameTag represents highlight field name for given field in search result, if value is empty then the highlight for this field is not used
// priority represents sort priority for given field
type FieldData struct {
	name     string
	nameTag  string
	priority float64
}

// GetName returns TriggerField name.
func (field FieldData) GetName() string {
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
