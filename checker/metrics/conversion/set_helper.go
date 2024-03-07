package conversion

var void struct{} = struct{}{}

// set[string] is a map that represents a set of strings with corresponding methods.
type set[K comparable] map[K]struct{}

func (set set[K]) contains(key K) bool {
	_, ok := set[key]
	return ok
}

func (set set[K]) insert(key K) {
	set[key] = void
}

func newSet[K comparable](value map[K]bool) set[K] {
	res := make(set[K], len(value))

	for k, v := range value {
		if v {
			res.insert(k)
		}
	}

	return res
}

// newSetFromTriggerTargetMetrics is a constructor function for setHelper.
func newSetFromTriggerTargetMetrics(metrics TriggerTargetMetrics) set[string] {
	result := make(set[string], len(metrics))
	for metricName := range metrics {
		result.insert(metricName)
	}
	return result
}

// diff is a set relative complement operation that returns a new set with elements.
// that appear only in second set.
func (self set[string]) diff(other set[string]) set[string] {
	result := make(set[string], len(self))

	for metricName := range other {
		if !self.contains(metricName) {
			result.insert(metricName)
		}
	}

	return result
}

// union is a sets union operation that return a new set with elements from both sets.
func (self set[string]) union(other set[string]) set[string] {
	result := make(set[string], len(self)+len(other))

	for metricName := range self {
		result.insert(metricName)
	}
	for metricName := range other {
		result.insert(metricName)
	}

	return result
}
