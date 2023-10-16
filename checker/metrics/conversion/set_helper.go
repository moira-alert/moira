package conversion

var void struct{} = struct{}{}

// set[string] is a map that represents a set of strings with corresponding methods.
type set[K comparable] map[K]struct{}

func (set set[K]) Contains(key K) bool {
	_, ok := set[key]
	return ok
}

func (set set[K]) Insert(key K) {
	set[key] = void
}

func NewSet[K comparable](value map[K]bool) set[K] {
	res := make(set[K], len(value))

	for k, v := range value {
		if v {
			res.Insert(k)
		}
	}

	return res
}

// newSetFromTriggerTargetMetrics is a constructor function for setHelper.
func newSetFromTriggerTargetMetrics(metrics TriggerTargetMetrics) set[string] {
	result := make(set[string], len(metrics))
	for metricName := range metrics {
		result[metricName] = void
	}
	return result
}

// diff is a set relative complement operation that returns a new set with elements
// that appear only in second set.
func (self set[string]) diff(other set[string]) set[string] {
	result := make(set[string], len(self))

	for metricName := range other {
		if !self.Contains(metricName) {
			result.Insert(metricName)
		}
	}

	return result
}

// union is a sets union operation that return a new set with elements from both sets.
func (self set[string]) union(other set[string]) set[string] {
	result := make(set[string], len(self)+len(other))

	for metricName := range self {
		result.Insert(metricName)
	}
	for metricName := range other {
		result.Insert(metricName)
	}

	return result
}
