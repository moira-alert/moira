package stats

import (
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/metrics"
)

type contactStats struct {
	metrics  *metrics.ContactsMetrics
	database moira.Database
	logger   moira.Logger
}

// NewContactStats creates and initializes a new contactStats object.
func NewContactStats(
	metricsRegistry metrics.Registry,
	database moira.Database,
	logger moira.Logger,
) *contactStats {
	return &contactStats{
		metrics:  metrics.NewContactsMetrics(metricsRegistry),
		database: database,
		logger:   logger,
	}
}

// StartReport starts reporting statistics about contacts.
func (stats *contactStats) StartReport(stop <-chan struct{}) {
	checkTicker := time.NewTicker(time.Minute)
	defer checkTicker.Stop()

	stats.logger.Info().Msg("Start contact statistics reporter")

	for {
		select {
		case <-stop:
			stats.logger.Info().Msg("Stop contact statistics reporter")
			return

		case <-checkTicker.C:
			stats.checkContactsCount()
		}
	}
}

func (stats *contactStats) checkContactsCount() {
	contacts, err := stats.database.GetAllContacts()
	if err != nil {
		stats.logger.Warning().
			Error(err).
			Msg("Failed to get all contacts")
		return
	}

	contactsCounter := make(map[string]int64)

	for _, contact := range contacts {
		if contact != nil {
			contactsCounter[contact.Type]++
		}
	}

	for contact, count := range contactsCounter {
		stats.metrics.Mark(contact, count)
	}
}
