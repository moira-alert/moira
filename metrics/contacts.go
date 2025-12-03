package metrics

import "regexp"

var nonAllowedMetricCharsRegex = regexp.MustCompile("[^a-zA-Z0-9_]")

// ContactsMetrics Collection of metrics for contacts counting.
type ContactsMetrics struct {
	contactsCount      map[string]Meter
	registry           Registry
	attributedRegistry MetricRegistry
}

// NewContactsMetrics Creates and configurates the instance of ContactsMetrics.
func NewContactsMetrics(registry Registry, attributedRegistry MetricRegistry) *ContactsMetrics {
	meters := make(map[string]Meter)

	return &ContactsMetrics{
		contactsCount:      meters,
		registry:           registry,
		attributedRegistry: attributedRegistry,
	}
}

// replaceNonAllowedMetricCharacters replaces non-allowed characters in the given metric string with underscores.
func (metrics *ContactsMetrics) replaceNonAllowedMetricCharacters(metric string) string {
	return nonAllowedMetricCharsRegex.ReplaceAllString(metric, "_")
}

// Mark Marks the number of contacts of different types.
func (metrics *ContactsMetrics) Mark(contact string, count int64) error {
	if _, ok := metrics.contactsCount[contact]; !ok {
		metric := metrics.replaceNonAllowedMetricCharacters(contact)
		attributedRegistry := metrics.attributedRegistry.WithAttributes(Attributes{
			Attribute{Key: "contact_type", Value: metric},
		})

		attributedGauge, err := attributedRegistry.NewGauge("contacts")
		if err != nil {
			return err
		}

		metrics.contactsCount[contact] = NewCompositeMeter(metrics.registry.NewMeter("contacts", metric), attributedGauge)
	}

	metrics.contactsCount[contact].Mark(count)

	return nil
}
