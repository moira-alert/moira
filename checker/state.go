package checker

// Default moira triggers states
const (
	OK        = "OK"
	WARN      = "WARN"
	ERROR     = "ERROR"
	NODATA    = "NODATA"
	EXCEPTION = "EXCEPTION"
	DEL       = "DEL"
)

func toMetricState(state string) string {
	if state == DEL {
		return NODATA
	}
	return state
}

// ToTriggerState is an auxiliary function to handle trigger state properly.
func ToTriggerState(state string) string {
	if state == DEL {
		return OK
	}
	return state
}
