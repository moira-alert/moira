package index

import (
	"os"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search/query"
	"github.com/moira-alert/moira"
)

const indexName = "moira-search-index.bleve"

// Worker represents Worker for Bleve.Index type
type Worker struct {
	index      bleve.Index
	logger     moira.Logger
	database   moira.Database
	inProgress bool
	indexed    bool
}

// NewSearchWorker return new Worker object
func NewSearchWorker(logger moira.Logger, database moira.Database) *Worker {
	return &Worker{
		logger:   logger,
		database: database,
	}
}

// Init initializes index. It removes old index files, create new mapping and index all triggers from database
func (worker *Worker) Init() error {
	if worker.inProgress {
		return nil
	}
	err := worker.createIndex()
	if err != nil {
		return err
	}
	err = worker.fillIndex()
	if err == nil {
		worker.indexed = true
		worker.inProgress = false
	}
	return err
}

// IsReady returns boolean value which determines if index is ready to use
func (worker *Worker) IsReady() bool {
	return worker.indexed
}

// Search search for triggers in index and returns slice of trigger IDs
func (worker *Worker) Search(filterTags, searchTerms []string) ([]string, error) {
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
	docs, _ := worker.index.DocCount()
	req.Size = int(docs)
	searchResult, err := worker.index.Search(req)
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
func (worker *Worker) Update(triggerIDs ...string) error {
	if len(triggerIDs) == 0 {
		return nil
	}
	worker.logger.Debugf("Update index for %d trigger IDs", len(triggerIDs))
	triggerChecks, err := worker.database.GetTriggerChecks(triggerIDs)
	if err != nil {
		return err
	}
	worker.logger.Debugf("Get %d trigger checks from DB", len(triggerChecks))
	for _, triggerCheck := range triggerChecks {
		if triggerCheck != nil {
			err = worker.index.Index(triggerCheck.ID, triggerCheck)
			if err != nil {
				return err
			}
			worker.logger.Debugf("Updated index for trigger ID %s", triggerCheck.ID)
		}
	}
	return nil
}

// Delete removes triggerIDs from index
func (worker *Worker) Delete(triggerIDs ...string) error {
	if len(triggerIDs) == 0 {
		return nil
	}
	for _, triggerID := range triggerIDs {
		worker.index.Delete(triggerID)
	}
	return nil
}

func (worker *Worker) createIndex() error {
	worker.logger.Debugf("Removing old index files: %s", indexName)
	destroyIndex(indexName)

	worker.logger.Debugf("Create new index")
	var err error
	worker.index, err = getIndex(indexName)
	return err
}

func (worker *Worker) fillIndex() error {
	worker.logger.Debugf("Start filling index with triggers: %s", indexName)
	worker.inProgress = true
	allTriggerIDs, err := worker.database.GetTriggerIDs()
	worker.logger.Debugf("Triggers IDs fetched from database: %d", len(allTriggerIDs))
	if err != nil {
		return err
	}

	allTriggersChecks, err := worker.database.GetTriggerChecks(allTriggerIDs)
	worker.logger.Debugf("Triggers checks fetched from database: %d", len(allTriggersChecks))
	if err != nil {
		return err
	}
	count, err := worker.addTriggers(allTriggersChecks)
	worker.logger.Infof("%d triggers added to index", count)
	return err
}

func (worker *Worker) addTriggers(triggers []*moira.TriggerCheck) (count int, err error) {
	toIndex := len(triggers)
	batch := worker.index.NewBatch()
	batchSize := 1000
	firstIndexed := false

	for _, trigger := range triggers {
		if trigger != nil {
			// ToDo: this code works, but looks stupid. We have to find a reason why Bleve indexes first batch 1 minute
			if !firstIndexed {
				worker.index.Index(trigger.ID, createIndexedTriggerCheck(*trigger))
				firstIndexed = true
			}
			err = batch.Index(trigger.ID, createIndexedTriggerCheck(*trigger))
			if err != nil {
				return
			}
		}
		if batch.Size() >= batchSize {
			err = worker.index.Batch(batch)
			if err != nil {
				return
			}
			count += batch.Size()
			batch = worker.index.NewBatch()
			worker.logger.Debugf("[%d triggers of %d] added to index", count, toIndex)
		}
	}
	if batch.Size() > 0 {
		err = worker.index.Batch(batch)
		if err == nil {
			count += batch.Size()
			worker.logger.Debugf("[%d triggers of %d] added to index", count, toIndex)
		}
	}
	return
}

func getIndex(indexPath string) (bleve.Index, error) {

	bleveIdx, err := bleve.Open(indexPath)
	if err != nil {
		indexMapping := buildIndexMapping()
		bleveIdx, err = bleve.New(indexPath, indexMapping)
		if err != nil {
			return nil, err
		}
	}

	return bleveIdx, nil
}

func destroyIndex(path string) {
	os.RemoveAll(path)
}
