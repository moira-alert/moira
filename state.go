package moira

// State type describe all default moira triggers or metrics states.
type State string

// TTLState declares all ttl (NODATA) states, used if metric has no values for given interval (ttl).
type TTLState string

// SelfStateWorkerState declares a selfState worker states.
type SelfStateWorkerState string

// Moira notifier self-states.
const (
	SelfStateOK    = "OK"    // OK means notifier is healthy
	SelfStateERROR = "ERROR" // ERROR means notifier is stopped, admin intervention is required
)

// Moira selfstate worker states.
const (
	SelfStateWorkerOK    SelfStateWorkerState = "OK"    // OK means selfstate does not register any errors
	SelfStateWorkerWARN  SelfStateWorkerState = "WARN"  // WARN means selfstate caught an error but notified only moira's admins
	SelfStateWorkerERROR SelfStateWorkerState = "ERROR" // ERROR means selfstate caught an error and notifications sended to admins and users
)

// Moira notifier management actors.
const (
	SelfStateActorAutomatic = "AUTO"
	SelfStateActorTrigger   = "TRIGGER"
	SelfStateActorManual    = "MANUAL"
)

// Moira trigger and metric states.
var (
	StateOK        State = "OK"
	StateWARN      State = "WARN"
	StateERROR     State = "ERROR"
	StateNODATA    State = "NODATA"
	StateEXCEPTION State = "EXCEPTION" // Use this for trigger check unexpected errors
	StateTEST      State = "TEST"      // Use this only for test notifications
)

// Moira ttl states.
var (
	TTLStateOK     TTLState = "OK"
	TTLStateWARN   TTLState = "WARN"
	TTLStateERROR  TTLState = "ERROR"
	TTLStateNODATA TTLState = "NODATA"
	TTLStateDEL    TTLState = "DEL"
)

var (
	eventStatesPriority = [...]State{StateOK, StateWARN, StateERROR, StateNODATA, StateEXCEPTION, StateTEST}
	stateScores         = map[State]int64{
		StateOK:        0,
		StateWARN:      1,
		StateERROR:     100,    //nolint
		StateNODATA:    1000,   //nolint
		StateEXCEPTION: 100000, //nolint
	}
	eventStateWeight = map[State]int{
		StateOK:     0,
		StateWARN:   1,
		StateERROR:  100,   //nolint
		StateNODATA: 10000, //nolint
	}
)

// String is a simple Stringer implementation for State.
func (state State) String() string {
	return string(state)
}

// ToSelfState converts State to corresponding SelfState.
func (state State) ToSelfState() string {
	if state != StateOK {
		return SelfStateERROR
	}

	return SelfStateOK
}

// IsValid checks if  valid State.
func (state State) IsValid() bool {
	for _, allowedState := range eventStatesPriority {
		if state == allowedState {
			return true
		}
	}

	return false
}

// ToMetricState is an auxiliary function to handle metric state properly.
func (state TTLState) ToMetricState() State {
	if state == TTLStateDEL {
		return StateNODATA
	}

	return State(state)
}

// ToTriggerState is an auxiliary function to handle trigger state properly.
func (state TTLState) ToTriggerState() State {
	if state == TTLStateDEL {
		return StateOK
	}

	return State(state)
}
