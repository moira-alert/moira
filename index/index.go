package index

import (
	"os"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/analysis/analyzer/keyword"
	"github.com/blevesearch/bleve/analysis/analyzer/standard"
	"github.com/blevesearch/bleve/mapping"
	"github.com/moira-alert/moira"
	"gopkg.in/tomb.v2"
)

const indexName = "moira-index.bleve"

type SearchIndex struct {
	bleveIndex bleve.Index
	logger     moira.Logger
	database   moira.Database
	tomb       tomb.Tomb
	indexed    bool
}

type listOfTriggers struct {
	List []moira.Trigger `json:"list"`
}

func NewSearchIndex(logger moira.Logger, database moira.Database) *SearchIndex {
	return &SearchIndex{
		logger:   logger,
		database: database,
	}
}

func (index *SearchIndex) Start() error {
	err := index.CreateIndex()
	if err != nil {
		return err
	}
	err = index.FillIndex()
	return err
}

func (index *SearchIndex) CreateIndex() error {
	index.logger.Debugf("Removing old index files: %s", indexName)
	destroyIndex(indexName)

	index.logger.Debugf("Create new index")
	var err error
	index.bleveIndex, err = getIndex(indexName)
	if err != nil {
		return err
	}

	return nil
}

func (index *SearchIndex) FillIndex() error {
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
	if err != nil {
		return err
	}

	return nil
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

func (index *SearchIndex) Stop() error {
	index.tomb.Kill(nil)
	destroyIndex(indexName)
	return index.tomb.Wait()
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

	// return de index
	return bleveIdx, nil
}

func buildIndexMapping() mapping.IndexMapping {

	// a generic reusable mapping for english text
	standardFieldMapping := bleve.NewTextFieldMapping()
	standardFieldMapping.Analyzer = standard.Name

	// a generic reusable mapping for keyword text
	keywordFieldMapping := bleve.NewTextFieldMapping()
	keywordFieldMapping.Analyzer = keyword.Name

	triggerMapping := bleve.NewDocumentMapping()

	triggerMapping.AddFieldMappingsAt("name", standardFieldMapping)
	triggerMapping.AddFieldMappingsAt("description", standardFieldMapping)
	triggerMapping.AddFieldMappingsAt("tags", keywordFieldMapping)

	indexMapping := bleve.NewIndexMapping()
	indexMapping.AddDocumentMapping(moira.Trigger{}.Type(), triggerMapping)
	return indexMapping
}

func destroyIndex(path string) {
	os.RemoveAll(path)
}
