package undefined

import "github.com/moira-alert/moira"

// GetDefaultProtector returns default Nodata protector
func GetDefaultProtector() *Protector {
	return &Protector{}
}

// Protector implements NoData Protector interface
type Protector struct{}

// GetStream returns stream of ProtectorData
func (protector *Protector) GetStream() <-chan moira.ProtectorData {
	ch := make(chan moira.ProtectorData)
	go func() {
		for {
			protectorData := &ProtectorData{}
			ch <- protectorData
		}
	}()
	return ch
}

// Protect performs Nodata protection
func (protector *Protector) Protect(protectorData moira.ProtectorData) error {
	return nil
}
