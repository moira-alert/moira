package graphite

import "time"

// MetersCollection implements meter collection abstraction
type MetersCollection interface {
	RegisterMeter(name, path string)
	GetRegisteredMeter(name string) (Meter, bool)
}

// Meter count events to produce exponentially-weighted moving average rates
// at one-, five-, and fifteen-minutes and a mean rate.
type Meter interface {
	Count() int64
	Mark(int64)
}

// Timer capture the duration and rate of events.
type Timer interface {
	Count() int64
	UpdateSince(time.Time)
}

// Histogram calculate distribution statistics from a series of int64 values.
type Histogram interface {
	Count() int64
	Update(int64)
}

// Counter hold an int64 value that can be incremented and decremented.
type Counter interface {
	Count() int64
	Inc(int64)
}
