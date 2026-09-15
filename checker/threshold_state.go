package checker

import "github.com/moira-alert/moira"

// thresholdState tracks a single WarnFor/ErrorFor timer: its configuration together with the
// bookkeeping that must be persisted between checks (MetricState's *Since/*RecoverSince fields).
type thresholdState struct {
	forDuration   int64
	keepFiringFor int64
	since         int64
	recoverSince  int64
}

// isFired reports whether the threshold has been continuously satisfied for at least forDuration.
func (t thresholdState) isFired(timestamp int64) bool {
	return t.since != 0 && timestamp-t.since >= t.forDuration
}

// advance moves the threshold state one check step forward and returns the updated state.
func (t thresholdState) advance(condition bool, timestamp int64) thresholdState {
	if condition {
		if t.since == 0 {
			t.since = timestamp
		}

		// Condition holds again, so cancel any grace period that was counting down.
		t.recoverSince = 0

		return t
	}

	if t.since == 0 {
		return t
	}

	if !t.isFired(timestamp) {
		// Never fired, so it resets immediately - no grace period applies.
		t.since = 0
		t.recoverSince = 0

		return t
	}

	// Already fired: anchor the recovery grace period on the first tick the condition drops.
	if t.recoverSince == 0 {
		t.recoverSince = timestamp
	}

	graceElapsed := t.keepFiringFor <= 0 || timestamp-t.recoverSince >= t.keepFiringFor
	if graceElapsed {
		t.since = 0
		t.recoverSince = 0
	}

	return t
}

// evaluateThresholds implements the WarnFor/ErrorFor/WarnKeepFiringFor/ErrorKeepFiringFor
// semantics. It must only be called with a raw state of OK, WARN or ERROR; NODATA/EXCEPTION are
// handled by the caller. A severity whose For/KeepFiringFor are both zero is left zero-valued
// instead of advanced, since it would fire on the same tick the raw condition becomes true anyway.
func evaluateThresholds(
	trigger *moira.Trigger,
	rawState moira.State,
	timestamp int64,
	prev moira.MetricState,
) (state moira.State, warnThreshold, errorThreshold thresholdState) {
	isWarnFired := rawState == moira.StateWARN || rawState == moira.StateERROR
	if trigger.WarnFor != 0 || trigger.WarnKeepFiringFor != 0 {
		warnThreshold = thresholdState{
			forDuration:   trigger.WarnFor,
			keepFiringFor: trigger.WarnKeepFiringFor,
			since:         prev.WarnSince,
			recoverSince:  prev.WarnRecoverSince,
		}.advance(isWarnFired, timestamp)

		isWarnFired = warnThreshold.isFired(timestamp)
	}

	isErrorFired := rawState == moira.StateERROR
	if trigger.ErrorFor != 0 || trigger.ErrorKeepFiringFor != 0 {
		errorThreshold = thresholdState{
			forDuration:   trigger.ErrorFor,
			keepFiringFor: trigger.ErrorKeepFiringFor,
			since:         prev.ErrorSince,
			recoverSince:  prev.ErrorRecoverSince,
		}.advance(isErrorFired, timestamp)

		isErrorFired = errorThreshold.isFired(timestamp)
	}

	state = moira.StateOK

	switch {
	case isErrorFired:
		state = moira.StateERROR
	case isWarnFired:
		state = moira.StateWARN
	}

	return state, warnThreshold, errorThreshold
}
