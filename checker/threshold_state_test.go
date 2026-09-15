package checker

import (
	"testing"

	"github.com/moira-alert/moira"
	"github.com/stretchr/testify/require"
)

func TestThresholdAdvance(t *testing.T) {
	t.Run("Condition met", func(t *testing.T) {
		t.Run("Was inactive, anchors since to now", func(t *testing.T) {
			result := thresholdState{forDuration: 10}.advance(true, 100)
			require.Equal(t, int64(100), result.since)
			require.Equal(t, int64(0), result.recoverSince)
			require.False(t, result.isFired(100))
		})

		t.Run("Already ticking, keeps original anchor", func(t *testing.T) {
			result := thresholdState{forDuration: 10, since: 100}.advance(true, 105)
			require.Equal(t, int64(100), result.since)
			require.Equal(t, int64(0), result.recoverSince)
			require.False(t, result.isFired(105))
		})

		t.Run("forDuration elapsed, fires", func(t *testing.T) {
			result := thresholdState{forDuration: 10, since: 100}.advance(true, 110)
			require.Equal(t, int64(100), result.since)
			require.Equal(t, int64(0), result.recoverSince)
			require.True(t, result.isFired(110))
		})

		t.Run("forDuration = 0 fires the same tick it becomes true", func(t *testing.T) {
			result := thresholdState{}.advance(true, 100)
			require.Equal(t, int64(100), result.since)
			require.True(t, result.isFired(100))
		})

		t.Run("Cancels an in-progress keep_firing_for countdown", func(t *testing.T) {
			result := thresholdState{
				forDuration: 10, keepFiringFor: 20, since: 100, recoverSince: 140,
			}.advance(true, 150)
			require.Equal(t, int64(100), result.since)
			require.Equal(t, int64(0), result.recoverSince)
			require.True(t, result.isFired(150))
		})
	})

	t.Run("Condition not met", func(t *testing.T) {
		t.Run("Already inactive, no-op", func(t *testing.T) {
			result := thresholdState{forDuration: 10}.advance(false, 100)
			require.Equal(t, int64(0), result.since)
			require.Equal(t, int64(0), result.recoverSince)
			require.False(t, result.isFired(100))
		})

		t.Run("Before forDuration elapses resets immediately, no grace", func(t *testing.T) {
			result := thresholdState{
				forDuration: 10, keepFiringFor: 100, since: 100,
			}.advance(false, 105)
			require.Equal(t, int64(0), result.since)
			require.Equal(t, int64(0), result.recoverSince)
			require.False(t, result.isFired(105))
		})

		t.Run("After firing, no keepFiringFor configured resolves immediately", func(t *testing.T) {
			result := thresholdState{forDuration: 10, since: 100}.advance(false, 111)
			require.Equal(t, int64(0), result.since)
			require.Equal(t, int64(0), result.recoverSince)
			require.False(t, result.isFired(111))
		})

		t.Run("After firing, within keepFiringFor grace stays fired", func(t *testing.T) {
			result := thresholdState{
				forDuration: 10, keepFiringFor: 20, since: 100,
			}.advance(false, 111)
			require.Equal(t, int64(100), result.since)
			require.Equal(t, int64(111), result.recoverSince)
			require.True(t, result.isFired(111))
		})

		t.Run("Grace already ticking, keeps original recover anchor", func(t *testing.T) {
			result := thresholdState{
				forDuration: 10, keepFiringFor: 20, since: 100, recoverSince: 111,
			}.advance(false, 120)
			require.Equal(t, int64(100), result.since)
			require.Equal(t, int64(111), result.recoverSince)
			require.True(t, result.isFired(120))
		})

		t.Run("Grace elapsed, resolves", func(t *testing.T) {
			result := thresholdState{
				forDuration: 10, keepFiringFor: 20, since: 100, recoverSince: 111,
			}.advance(false, 131)
			require.Equal(t, int64(0), result.since)
			require.Equal(t, int64(0), result.recoverSince)
			require.False(t, result.isFired(131))
		})
	})
}

func TestEvaluateThresholds(t *testing.T) {
	warnValue, errorValue := 10.0, 20.0
	baseTrigger := func() *moira.Trigger {
		return &moira.Trigger{
			WarnValue:   &warnValue,
			ErrorValue:  &errorValue,
			TriggerType: moira.RisingTrigger,
		}
	}

	t.Run("For = 0 / unset behaves exactly like instant firing/resolving, without persisting timer bookkeeping", func(t *testing.T) {
		trigger := baseTrigger()

		state, warnThreshold, errorThreshold := evaluateThresholds(trigger, moira.StateWARN, 100, moira.MetricState{})
		require.Equal(t, moira.StateWARN, state)
		require.Equal(t, thresholdState{}, warnThreshold)
		require.Equal(t, thresholdState{}, errorThreshold)

		// Prior WarnSince is ignored too: with WarnFor/WarnKeepFiringFor both unset, the warn
		// threshold is never tracked at all, regardless of what MetricState carried over.
		state, warnThreshold, errorThreshold = evaluateThresholds(trigger, moira.StateERROR, 110, moira.MetricState{WarnSince: 100})
		require.Equal(t, moira.StateERROR, state)
		require.Equal(t, thresholdState{}, warnThreshold)
		require.Equal(t, thresholdState{}, errorThreshold)

		state, warnThreshold, errorThreshold = evaluateThresholds(trigger, moira.StateOK, 120, moira.MetricState{WarnSince: 100, ErrorSince: 110})
		require.Equal(t, moira.StateOK, state)
		require.Equal(t, thresholdState{}, warnThreshold)
		require.Equal(t, thresholdState{}, errorThreshold)
	})

	t.Run("Motivating scenario: WarnFor=10s, ErrorFor=10s, 5s WARN band then 5s ERROR band", func(t *testing.T) {
		trigger := baseTrigger()
		trigger.WarnFor = 10
		trigger.ErrorFor = 10

		// Base offset keeps every "since" timestamp non-zero, since 0 doubles as the
		// "not currently tracked" sentinel in the algorithm.
		const base int64 = 1000

		metricState := moira.MetricState{}

		// base+0..base+4: WARN band, WarnFor timer starts ticking, not fired yet.
		for now := base; now <= base+4; now++ {
			state, warnThreshold, errorThreshold := evaluateThresholds(trigger, moira.StateWARN, now, metricState)
			require.Equal(t, moira.StateOK, state)

			metricState.WarnSince, metricState.WarnRecoverSince = warnThreshold.since, warnThreshold.recoverSince
			metricState.ErrorSince, metricState.ErrorRecoverSince = errorThreshold.since, errorThreshold.recoverSince
		}

		require.Equal(t, base, metricState.WarnSince)

		// base+5..base+9: ERROR band. WarnFor keeps accumulating from the original base+0 anchor
		// (never reset by the escalation); ErrorFor starts its own timer at base+5.
		for now := base + 5; now <= base+9; now++ {
			state, warnThreshold, errorThreshold := evaluateThresholds(trigger, moira.StateERROR, now, metricState)
			require.Equal(t, moira.StateOK, state)

			metricState.WarnSince = warnThreshold.since
			metricState.ErrorSince = errorThreshold.since
		}

		require.Equal(t, base, metricState.WarnSince)
		require.Equal(t, base+5, metricState.ErrorSince)

		// base+10: WARN's timer (anchored at base+0) reaches exactly WarnFor=10s -> WARN fires.
		// ERROR's timer (anchored at base+5) has only accumulated 5s -> not fired yet.
		state, warnThreshold, errorThreshold := evaluateThresholds(trigger, moira.StateERROR, base+10, metricState)
		require.Equal(t, moira.StateWARN, state)

		metricState.WarnSince = warnThreshold.since
		metricState.ErrorSince = errorThreshold.since

		// base+11..base+14: still in ERROR band, ERROR timer keeps accumulating.
		for now := base + 11; now <= base+14; now++ {
			state, _, errorThreshold := evaluateThresholds(trigger, moira.StateERROR, now, metricState)
			require.Equal(t, moira.StateWARN, state)

			metricState.ErrorSince = errorThreshold.since
		}

		// base+15: ERROR's timer (anchored at base+5) reaches ErrorFor=10s -> escalates to ERROR.
		state, _, _ = evaluateThresholds(trigger, moira.StateERROR, base+15, metricState)
		require.Equal(t, moira.StateERROR, state)
	})

	t.Run("Full reset when the metric returns to OK before its threshold's for elapses", func(t *testing.T) {
		trigger := baseTrigger()
		trigger.WarnFor = 10

		state, warnThreshold, _ := evaluateThresholds(trigger, moira.StateWARN, 100, moira.MetricState{})
		require.Equal(t, moira.StateOK, state)

		state, warnThreshold, _ = evaluateThresholds(trigger, moira.StateOK, 105, moira.MetricState{
			WarnSince:        warnThreshold.since,
			WarnRecoverSince: warnThreshold.recoverSince,
		})
		require.Equal(t, moira.StateOK, state)
		require.Equal(t, int64(0), warnThreshold.since)
		require.Equal(t, int64(0), warnThreshold.recoverSince)
	})

	t.Run("WarnKeepFiringFor: brief dip below threshold doesn't resolve within grace, resolves after", func(t *testing.T) {
		trigger := baseTrigger()
		trigger.WarnKeepFiringFor = 20

		metricState := moira.MetricState{}
		state, warnThreshold, _ := evaluateThresholds(trigger, moira.StateWARN, 100, metricState)
		require.Equal(t, moira.StateWARN, state)

		metricState.WarnSince, metricState.WarnRecoverSince = warnThreshold.since, warnThreshold.recoverSince

		// dip below threshold, within grace
		state, warnThreshold, _ = evaluateThresholds(trigger, moira.StateOK, 110, metricState)
		require.Equal(t, moira.StateWARN, state)

		metricState.WarnSince, metricState.WarnRecoverSince = warnThreshold.since, warnThreshold.recoverSince

		// grace elapses (started at t=110, WarnKeepFiringFor=20 -> elapses at t=130)
		state, warnThreshold, _ = evaluateThresholds(trigger, moira.StateOK, 131, metricState)
		require.Equal(t, moira.StateOK, state)
		require.Equal(t, int64(0), warnThreshold.since)
		require.Equal(t, int64(0), warnThreshold.recoverSince)
	})

	t.Run("ErrorKeepFiringFor: downgrade ERROR to WARN (not OK) once grace expires while WARN condition still true", func(t *testing.T) {
		trigger := baseTrigger()
		trigger.ErrorKeepFiringFor = 20

		metricState := moira.MetricState{}
		state, warnThreshold, errorThreshold := evaluateThresholds(trigger, moira.StateERROR, 100, metricState)
		require.Equal(t, moira.StateERROR, state)

		metricState.WarnSince = warnThreshold.since
		metricState.ErrorSince, metricState.ErrorRecoverSince = errorThreshold.since, errorThreshold.recoverSince

		// drops to WARN band: WARN condition true, ERROR condition false -> ERROR grace starts.
		state, warnThreshold, errorThreshold = evaluateThresholds(trigger, moira.StateWARN, 110, metricState)
		require.Equal(t, moira.StateERROR, state) // still within ErrorKeepFiringFor grace

		metricState.WarnSince = warnThreshold.since
		metricState.ErrorSince, metricState.ErrorRecoverSince = errorThreshold.since, errorThreshold.recoverSince

		// grace elapses (started at t=110, ErrorKeepFiringFor=20 -> elapses at t=130) while WARN
		// condition is still true -> downgrades to WARN, not OK.
		state, _, errorThreshold = evaluateThresholds(trigger, moira.StateWARN, 131, metricState)
		require.Equal(t, moira.StateWARN, state)
		require.Equal(t, int64(0), errorThreshold.since)
		require.Equal(t, int64(0), errorThreshold.recoverSince)
	})
}
