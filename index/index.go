package index

import (
	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search/query"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/index/mapping"
	"gopkg.in/tomb.v2"
)

// Index represents Index for Bleve.Index type
type Index struct {
	index      bleve.Index
	logger     moira.Logger
	database   moira.Database
	tomb       tomb.Tomb
	inProgress bool
	indexed    bool
}

// NewSearchIndex return new Index object
func NewSearchIndex(logger moira.Logger, database moira.Database) *Index {
	return &Index{
		logger:   logger,
		database: database,
	}
}

// Start initializes index. It removes old index files, create new mapping and index all triggers from database
func (index *Index) Start() error {
	if index.inProgress {
		return nil
	}
	err := index.createIndex()
	if err != nil {
		return err
	}
	err = index.fillIndex()
	if err == nil {
		index.indexed = true
		index.inProgress = false
	}

	index.tomb.Go(index.actualizeIndex)
	return err
}

// IsReady returns boolean value which determines if index is ready to use
func (index *Index) IsReady() bool {
	return index.indexed
}

// Search search for triggers in index and returns slice of trigger IDs
func (index *Index) Search(filterTags, searchTerms []string) ([]string, error) {
	searchQueries := make([]query.Query, 0)

	for _, tag := range filterTags {
		qr := bleve.NewTermQuery(tag)
		qr.FieldVal = "Tags"
		searchQueries = append(searchQueries, qr)
	}

	for _, term := range searchTerms {
		qr := bleve.NewFuzzyQuery(term)
		searchQueries = append(searchQueries, qr)
	}

	searchQuery := bleve.NewConjunctionQuery(searchQueries...)
	req := bleve.NewSearchRequest(searchQuery)
	docs, _ := index.index.DocCount()
	req.Size = int(docs)
	searchResult, err := index.index.Search(req)
	if err != nil {
		return []string{}, err
	}
	if searchResult.Hits.Len() == 0 {
		return []string{}, nil
	}
	triggerIds := make([]string, 0)
	for _, result := range searchResult.Hits {
		triggerIds = append(triggerIds, result.ID)
	}
	return triggerIds, nil
}

// Update get triggerIDs and updates it's in index
func (index *Index) Update(triggerIDs ...string) error {
	if len(triggerIDs) == 0 {
		return nil
	}
	index.logger.Debugf("Update index for %d trigger IDs", len(triggerIDs))
	triggerChecks, err := index.database.GetTriggerChecks(triggerIDs)
	if err != nil {
		return err
	}
	index.logger.Debugf("Get %d trigger checks from DB", len(triggerChecks))
	for _, triggerCheck := range triggerChecks {
		if triggerCheck != nil {
			err = index.index.Index(triggerCheck.ID, triggerCheck)
			if err != nil {
				return err
			}
			index.logger.Debugf("Updated index for trigger ID %s", triggerCheck.ID)
		}
	}
	return nil
}

// Delete removes triggerIDs from index
func (index *Index) Delete(triggerIDs ...string) error {
	if len(triggerIDs) == 0 {
		return nil
	}
	for _, triggerID := range triggerIDs {
		index.index.Delete(triggerID)
	}
	return nil
}

func (index *Index) createIndex() error {
	index.logger.Infof("Create new index for full-text search")
	var err error
	indexMapping := mapping.BuildIndexMapping(mapping.Trigger{})
	index.index, err = buildIndex(indexMapping)
	return err
}

func (index *Index) fillIndex() error {
	index.logger.Debugf("Start filling index with triggers")
	index.inProgress = true
	allTriggerIDs, err := index.database.GetTriggerIDs()
	index.logger.Debugf("Triggers IDs fetched from database: %d", len(allTriggerIDs))
	if err != nil {
		return err
	}

	count, err := index.addTriggers(allTriggerIDs)
	index.logger.Infof("%d triggers added to index", count)
	return err
}

func (index *Index) addTriggers(triggerIDs []string) (count int, err error) {
	toIndex := len(triggerIDs)
	batch := index.index.NewBatch()
	batchSize := 1000
	firstIndexed := false

	triggerIDsBatches := moira.ChunkSlice(triggerIDs, batchSize)
	var triggersToCheck []*moira.TriggerCheck

	for _, triggerIDsBatch := range triggerIDsBatches {
		triggersToCheck, err = index.database.GetTriggerChecks(triggerIDsBatch)
		index.logger.Debugf("Get %d trigger checks from DB", len(triggersToCheck))
		if err != nil {
			return
		}
		for _, trigger := range triggersToCheck {
			if trigger != nil {
				// ToDo: this code works, but looks stupid. We have to find a reason why Bleve indexes first batch 1 minute
				if !firstIndexed {
					index.index.Index(trigger.ID, mapping.CreateIndexedTrigger(*trigger))
					firstIndexed = true
				}
				err = batch.Index(trigger.ID, mapping.CreateIndexedTrigger(*trigger))
				if err != nil {
					return
				}
			}
		}
		err = index.index.Batch(batch)
		if err != nil {
			return
		}
		count += batch.Size()
		batch.Reset()
		index.logger.Debugf("[%d triggers of %d] added to index", count, toIndex)
	}
	return
}

// Stop stops checks triggers
func (index *Index) Stop() error {
	index.tomb.Kill(nil)
	return index.tomb.Wait()
}
