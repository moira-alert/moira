package metrics

import (
	"sync"
	"time"
)

// Registry implements metrics collection abstraction.
type Registry interface {
	NewMeter(path ...string) Meter
	NewTimer(path ...string) Timer
	NewHistogram(path ...string) Histogram
	NewCounter(path ...string) Counter
}

// MetersCollection implements meter collection abstraction.
type MetersCollection interface {
	RegisterMeter(name string, path ...string) Meter
	GetRegisteredMeter(name string) (Meter, bool)
}

// Meter count events to produce exponentially-weighted moving average rates.
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
	Inc()
}

func NewMetersCollection(registry Registry) MetersCollection {
	return &DefaultMetersCollection{registry: registry, meters: map[string]Meter{}}
}

// DefaultMetersCollection holds registered meters.
type DefaultMetersCollection struct {
	registry Registry
	meters   map[string]Meter
	mutex    sync.Mutex
}

func (source *DefaultMetersCollection) RegisterMeter(name string, path ...string) Meter {
	source.mutex.Lock()
	defer source.mutex.Unlock()

	source.meters[name] = source.registry.NewMeter(path...)
	return source.meters[name]
}

func (source *DefaultMetersCollection) GetRegisteredMeter(name string) (Meter, bool) {
	source.mutex.Lock()
	defer source.mutex.Unlock()

	value, found := source.meters[name]
	return value, found
}
