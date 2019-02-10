package moira

type State string

type TtlState string

// Default moira triggers and metrics states
const (
	StateOK     State = "OK"
	StateWARN   State = "WARN"
	StateERROR  State = "ERROR"
	StateNODATA State = "NODATA"
	// Use this for trigger check unexpected errors
	StateEXCEPTION State = "EXCEPTION"
	// Use this only for test notifications
	StateTEST State = "TEST"
)

// Moira ttl states
const (
	TTLStateOK     TtlState = "OK"
	TTLStateWARN   TtlState = "WARN"
	TTLStateERROR  TtlState = "ERROR"
	TTLStateNODATA TtlState = "NODATA"
	TTLStateDEL    TtlState = "DEL"
)

var (
	eventStatesPriority = [...]State{StateOK, StateWARN, StateERROR, StateNODATA, StateEXCEPTION, StateTEST}
	stateScore          = map[State]int64{
		StateOK:        0,
		StateWARN:      1,
		StateERROR:     100,
		StateNODATA:    1000,
		StateEXCEPTION: 100000,
	}
	eventStateWeight1 = map[State]int{
		StateOK:     0,
		StateWARN:   1,
		StateERROR:  100,
		StateNODATA: 10000,
	}
)

func (state TtlState) ToMetricState() State {
	if state == TTLStateDEL {
		return StateNODATA
	}
	return State(state)
}

// ToTriggerState is an auxiliary function to handle trigger state properly.
func (state TtlState) ToTriggerState() State {
	if state == TTLStateDEL {
		return StateOK
	}
	return State(state)
}
