package metrics

import (
	"time"
)

type CompositeRegistry struct {
	registries []Registry
}

func NewCompositeRegistry(registries ...Registry) *CompositeRegistry {
	return &CompositeRegistry{registries}
}

func (source *CompositeRegistry) NewMeter(path ...string) Meter {
	meters := make([]Meter, 0)
	for _, registry := range source.registries {
		meters = append(meters, registry.NewMeter(path...))
	}

	return &compositeMeter{meters}
}

func (source *CompositeRegistry) NewTimer(path ...string) Timer {
	timers := make([]Timer, 0)
	for _, registry := range source.registries {
		timers = append(timers, registry.NewTimer(path...))
	}

	return &compositeTimer{timers}
}

func (source *CompositeRegistry) NewHistogram(path ...string) Histogram {
	histograms := make([]Histogram, 0)
	for _, registry := range source.registries {
		histograms = append(histograms, registry.NewHistogram(path...))
	}

	return &compositeHistogram{histograms}
}

func (source *CompositeRegistry) NewCounter(path ...string) Counter {
	counters := make([]Counter, 0)
	for _, registry := range source.registries {
		counters = append(counters, registry.NewCounter(path...))
	}

	return &compositeCounter{counters}
}

type compositeMeter struct {
	meters []Meter
}

func NewCompositeMeter(meters ...Meter) Meter {
	return &compositeMeter{
		meters: meters,
	}
}

func (source *compositeMeter) Mark(value int64) {
	for _, meter := range source.meters {
		meter.Mark(value)
	}
}

type compositeTimer struct {
	timers []Timer
}

func NewCompositeTimer(timers ...Timer) Timer {
	return &compositeTimer{
		timers: timers,
	}
}

func (source *compositeTimer) Count() int64 {
	if len(source.timers) == 0 {
		return 0
	}

	return source.timers[0].Count()
}

func (source *compositeTimer) UpdateSince(ts time.Time) {
	for _, timer := range source.timers {
		timer.UpdateSince(ts)
	}
}

type compositeHistogram struct {
	histograms []Histogram
}

func NewCompositeHistogram(histograms ...Histogram) Histogram {
	return &compositeHistogram{
		histograms: histograms,
	}
}

func (source *compositeHistogram) Update(value int64) {
	for _, histogram := range source.histograms {
		histogram.Update(value)
	}
}

type compositeCounter struct {
	counters []Counter
}

func NewCompositeCounter(counters ...Counter) Counter {
	return &compositeCounter{
		counters: counters,
	}
}

func (source *compositeCounter) Count() int64 {
	if len(source.counters) == 0 {
		return 0
	}

	return source.counters[0].Count()
}

func (source *compositeCounter) Inc() {
	for _, counter := range source.counters {
		counter.Inc()
	}
}
