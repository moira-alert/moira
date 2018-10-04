package index

import (
	"os"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search/query"
	"github.com/moira-alert/moira"
)

const indexName = "moira-search-index.bleve"

// SearchIndex represents Bleve.Index type
type SearchIndex struct {
	bleveIndex bleve.Index
	logger     moira.Logger
	database   moira.Database
	inProgress bool
	indexed    bool
}

// NewSearchIndex return new SearchIndex object
func NewSearchIndex(logger moira.Logger, database moira.Database) *SearchIndex {
	return &SearchIndex{
		logger:   logger,
		database: database,
	}
}

// Init initializes index. It removes old index files, create new mapping and index all triggers from database
func (index *SearchIndex) Init() error {
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
	return err
}

// IsReady returns boolean value which determines if index is ready to use
func (index *SearchIndex) IsReady() bool {
	return index.indexed
}

// FindTriggerIds search for triggers in index and returns slice of trigger IDs
func (index *SearchIndex) FindTriggerIds(filterTags, searchTerms []string) ([]string, error) {
	searchQueries := make([]query.Query, 0)

	for _, tag := range filterTags {
		qr := bleve.NewTermQuery(tag)
		qr.FieldVal = "tags"
		searchQueries = append(searchQueries, qr)
	}

	for _, term := range searchTerms {
		qr := bleve.NewFuzzyQuery(term)
		searchQueries = append(searchQueries, qr)
	}

	searchQuery := bleve.NewConjunctionQuery(searchQueries...)
	req := bleve.NewSearchRequest(searchQuery)
	searchResult, err := index.bleveIndex.Search(req)
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

func (index *SearchIndex) createIndex() error {
	index.logger.Debugf("Removing old index files: %s", indexName)
	destroyIndex(indexName)

	index.logger.Debugf("Create new index")
	var err error
	index.bleveIndex, err = getIndex(indexName)
	return err
}

func (index *SearchIndex) fillIndex() error {
	index.inProgress = true
	allTriggerIDs, err := index.database.GetTriggerIDs()
	if err != nil {
		return err
	}

	allTriggers, err := index.database.GetTriggers(allTriggerIDs)
	if err != nil {
		return err
	}
	count, err := index.addTriggers(allTriggers)
	index.logger.Infof("%d triggers added to index", count)
	return err
}

func (index *SearchIndex) addTriggers(triggers []*moira.Trigger) (count int, err error) {
	batch := index.bleveIndex.NewBatch()
	batchSize := 100

	i := 0

	for _, trigger := range triggers {
		if trigger != nil {
			err = batch.Index(trigger.ID, &trigger)
			if err != nil {
				return
			}
		}
		if i > batchSize {
			err = index.bleveIndex.Batch(batch)
			if err != nil {
				return
			}
			i = 0
			count += batchSize
		}
	}
	if batch.Size() > 0 {
		err = index.bleveIndex.Batch(batch)
		if err != nil {
			count += batch.Size()
		}
	}
	return count, nil
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
