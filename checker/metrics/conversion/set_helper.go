package conversion

// setHelper is a map that represents a set of strings with corresponding methods.
type setHelper map[string]bool

// newSetHelperFromTriggerTargetMetrics is a constructor function for setHelper.
func newSetHelperFromTriggerTargetMetrics(metrics TriggerTargetMetrics) setHelper {
	result := make(setHelper, len(metrics))
	for metricName := range metrics {
		result[metricName] = true
	}
	return result
}

// diff is a set relative complement operation that returns a new set with elements
// that appear only in second set.
func (h setHelper) diff(other setHelper) setHelper {
	result := make(setHelper, len(h))
	for metricName := range other {
		if _, ok := h[metricName]; !ok {
			result[metricName] = true
		}
	}
	return result
}

// union is a sets union operation that return a new set with elements from both sets.
func (h setHelper) union(other setHelper) setHelper {
	result := make(setHelper, len(h)+len(other))
	for metricName := range h {
		result[metricName] = true
	}
	for metricName := range other {
		result[metricName] = true
	}
	return result
}
