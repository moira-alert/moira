package plotting

import (
	"time"
)

// Limits is a set of limits for given metricsData
type Limits struct {
	From    time.Time
	To      time.Time
	Lowest  float64
	Highest float64
}

// FormsSetContaining returns true if dot can belong to a set formed from limit values
func (limits Limits) FormsSetContaining(dot float64) bool {
	if (dot >= limits.Lowest) && (dot <= limits.Highest) {
		return true
	}
	return false
}
