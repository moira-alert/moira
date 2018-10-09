package index

import (
	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/analysis/analyzer/keyword"
	"github.com/blevesearch/bleve/analysis/analyzer/standard"
	"github.com/blevesearch/bleve/mapping"
)

func buildIndexMapping() mapping.IndexMapping {

	// a generic reusable mapping for english text
	standardFieldMapping := bleve.NewTextFieldMapping()
	standardFieldMapping.Analyzer = standard.Name
	standardFieldMapping.Store = false

	// a generic reusable mapping for keyword text
	keywordFieldMapping := bleve.NewTextFieldMapping()
	keywordFieldMapping.Analyzer = keyword.Name
	keywordFieldMapping.Store = false

	triggerMapping := bleve.NewDocumentMapping()

	triggerMapping.AddFieldMappingsAt("name", standardFieldMapping)
	triggerMapping.AddFieldMappingsAt("desc", standardFieldMapping)
	triggerMapping.AddFieldMappingsAt("tags", keywordFieldMapping)

	indexMapping := bleve.NewIndexMapping()
	indexMapping.AddDocumentMapping(indexedTrigger{}.Type(), triggerMapping)
	return indexMapping
}
