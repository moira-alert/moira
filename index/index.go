package index

import (
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/index/bleve"
	"github.com/moira-alert/moira/index/mapping"
	"github.com/moira-alert/moira/metrics"
	"gopkg.in/tomb.v2"
)

const defaultIndexBatchSize = 1000

// TriggerIndex is index for moira.TriggerChecks type
type TriggerIndex interface {
	Search(filterTags []string, searchString string, onlyErrors bool, page int64, size int64) (searchResults []*moira.SearchResult, total int64, err error)
	Write(checks []*moira.TriggerCheck) error
	Delete(triggerIDs []string) error
	GetCount() (int64, error)
}

// Index represents Index for Bleve.Index type
type Index struct {
	triggerIndex      TriggerIndex
	logger            moira.Logger
	database          moira.Database
	tomb              tomb.Tomb
	metrics           *metrics.IndexMetrics
	inProgress        bool
	indexed           bool
	indexActualizedTS int64
}

// NewSearchIndex return new Index object
func NewSearchIndex(logger moira.Logger, database moira.Database, metricsRegistry metrics.Registry) *Index {
	var err error
	newIndex := Index{
		logger:   logger,
		database: database,
	}
	newIndex.metrics = metrics.ConfigureIndexMetrics(metricsRegistry)
	indexMapping := mapping.BuildIndexMapping(mapping.Trigger{})
	newIndex.triggerIndex, err = bleve.CreateTriggerIndex(indexMapping)
	if err != nil {
		return nil
	}
	return &newIndex
}

// Start initializes index. It creates new mapping and index all triggers from database
func (index *Index) Start() error {
	if index.inProgress || index.indexed {
		return nil
	}

	err := index.fillIndex()
	if err != nil {
		return err
	}

	index.indexed = true
	index.inProgress = false

	index.tomb.Go(index.runIndexActualizer)
	index.tomb.Go(index.runTriggersToReindexSweepper)
	index.tomb.Go(index.checkIndexActualizationLag)
	index.tomb.Go(index.checkIndexedTriggersCount)

	return nil
}

// IsReady returns boolean value which determines if index is ready to use
func (index *Index) IsReady() bool {
	return index.indexed
}

// Stop stops checks triggers
func (index *Index) Stop() error {
	index.logger.Info("Stop search index")
	index.tomb.Kill(nil)
	return index.tomb.Wait()
}
