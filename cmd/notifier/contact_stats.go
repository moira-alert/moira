package main

import (
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/metrics"
	"gopkg.in/tomb.v2"
)

type contactStats struct {
	tomb     tomb.Tomb
	metrics  *metrics.ContactsMetrics
	database moira.Database
	logger   moira.Logger
}

func newContactStats(
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

func (stats *contactStats) start() {
	stats.tomb.Go(stats.startCheckingContactsCount)
}

func (stats *contactStats) startCheckingContactsCount() error {
	checkTicker := time.NewTicker(time.Minute)
	defer checkTicker.Stop()

	for {
		select {
		case <-stats.tomb.Dying():
			return nil

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
		stats.metrics.Mark(metrics.ReplaceNonAllowedMetricCharacters(contact), count)
	}
}

func (stats *contactStats) stop() error {
	stats.tomb.Kill(nil)
	return stats.tomb.Wait()
}
