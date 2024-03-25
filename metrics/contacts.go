package metrics

// Collection of metrics for contacts counting.
type ContactsMetrics struct {
	contactsCount map[string]Meter
	registry      Registry
}

// Creates and configurates the instance of ContactsMetrics.
func NewContactsMetrics(registry Registry) *ContactsMetrics {
	meters := make(map[string]Meter)

	return &ContactsMetrics{
		contactsCount: meters,
		registry:      registry,
	}
}

// Marks the number of contacts of different types.
func (metrics *ContactsMetrics) Mark(contact string, count int64) {
	if _, ok := metrics.contactsCount[contact]; !ok {
		metrics.contactsCount[contact] = metrics.registry.NewMeter("contacts", contact)
	}

	metrics.contactsCount[contact].Mark(count)
}
