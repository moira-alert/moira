package datatypes

// HeartbeatType are Moira's special internal types of problems.
type HeartbeatType string

const (
	HeartbeatTypeNotSet    HeartbeatType = "Heartbeat_type_not_set"
	HeartbeatNotifier      HeartbeatType = "heartbeat_notifier"
	HeartbeatDatabase      HeartbeatType = "heartbeat_database"
	HeartbeatLocalChecker  HeartbeatType = "heartbeat_local_checker"
	HeartbeatRemoteChecker HeartbeatType = "heartbeat_remote_checker"
	HeartbeatFilter        HeartbeatType = "heartbeat_filter"
)

// IsValid checks if such an heartbeat type exists.
func (heartbeatType HeartbeatType) IsValid() bool {
	switch heartbeatType {
	case HeartbeatNotifier, HeartbeatDatabase, HeartbeatLocalChecker, HeartbeatRemoteChecker, HeartbeatFilter:
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
