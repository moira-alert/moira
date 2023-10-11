package conversion

var void struct{} = struct{}{}

// setᐸstringᐳ is a map that represents a set of strings with corresponding methods.
type setᐸstringᐳ map[string]struct{}

func (set setᐸstringᐳ) Contains(str string) bool {
	_, ok := set[str]
	return ok
}

func (set setᐸstringᐳ) Insert(str string) {
	set[str] = void
}

func NewSet(set map[string]bool) setᐸstringᐳ {
	res := make(setᐸstringᐳ, len(set))

	for k, v := range set {
		if v {
			res.Insert(k)
		}
	}

	return res
}

// newSetHelperFromTriggerTargetMetrics is a constructor function for setHelper.
func newSetHelperFromTriggerTargetMetrics(metrics TriggerTargetMetrics) setᐸstringᐳ {
	result := make(setᐸstringᐳ, len(metrics))
	for metricName := range metrics {
		result[metricName] = void
	}
	return result
}

// diff is a set relative complement operation that returns a new set with elements
// that appear only in second set.
func (self setᐸstringᐳ) diff(other setᐸstringᐳ) setᐸstringᐳ {
	result := make(setᐸstringᐳ, len(self))

	for metricName := range other {
		if !self.Contains(metricName) {
			result.Insert(metricName)
		}
	}

	return result
}

// union is a sets union operation that return a new set with elements from both sets.
func (self setᐸstringᐳ) union(other setᐸstringᐳ) setᐸstringᐳ {
	result := make(setᐸstringᐳ, len(self)+len(other))

	for metricName := range self {
		result.Insert(metricName)
	}
	for metricName := range other {
		result.Insert(metricName)
	}

	return result
}
