package matched

// Protector implements ProtectorData interface
type ProtectorData struct {
	values []float64
}

// GetFloats returns floats
func (protectorData *ProtectorData) GetFloats() []float64 {
	return protectorData.values
}
