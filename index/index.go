package index

import (
	"github.com/blevesearch/bleve"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/index/mapping"
	"gopkg.in/tomb.v2"
)

const defaultIndexBatchSize = 1000

// Index represents Index for Bleve.Index type
type Index struct {
	index             bleve.Index
	logger            moira.Logger
	database          moira.Database
	tomb              tomb.Tomb
	inProgress        bool
	indexed           bool
	indexActualizedTS int64
}

// NewSearchIndex return new Index object
func NewSearchIndex(logger moira.Logger, database moira.Database) *Index {
	var err error
	newIndex := Index{
		logger:   logger,
		database: database,
	}
	indexMapping := mapping.BuildIndexMapping(mapping.Trigger{})
	newIndex.index, err = buildIndex(indexMapping)
	if err != nil {
		return nil
	}
	return &newIndex
}

// Start initializes index. It removes old index files, create new mapping and index all triggers from database
func (index *Index) Start() error {
	if index.inProgress || index.indexed {
		return nil
	}

	err := index.fillIndex()
	if err == nil {
		index.indexed = true
		index.inProgress = false
	}

	index.tomb.Go(index.runIndexActualizer)
	index.tomb.Go(index.runTriggersToReindexSweepper)

	return err
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
