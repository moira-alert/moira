package datatypes

// HeartbeatType are Moira's special internal types of problems.
type HeartbeatType string

const (
	HeartbeatTypeNotSet  HeartbeatType = "type_not_set"
	HeartbeatNotifierOff HeartbeatType = "notifier_off"
)

// IsValid checks if such an heartbeat type exists.
func (heartbeatType HeartbeatType) IsValid() bool {
	switch heartbeatType {
	case HeartbeatNotifierOff:
		return true
	default:
		return false
	}
}

// EmergencyContact is the structure for contacts to which notifications will go in the event of special internal Moira problems.
type EmergencyContact struct {
	ContactID      string
	HeartbeatTypes []HeartbeatType
}
