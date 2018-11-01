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
	standardFieldMapping.IncludeTermVectors = false

	// a generic reusable mapping for keyword text
	keywordFieldMapping := bleve.NewTextFieldMapping()
	keywordFieldMapping.Analyzer = keyword.Name
	keywordFieldMapping.Store = false
	keywordFieldMapping.IncludeTermVectors = false
	keywordFieldMapping.IncludeInAll = false

	// a generic numeric mapping for digits
	numericFieldMapping := bleve.NewNumericFieldMapping()

	triggerMapping := bleve.NewDocumentStaticMapping()

	triggerMapping.AddFieldMappingsAt("Name", standardFieldMapping)
	triggerMapping.AddFieldMappingsAt("Desc", standardFieldMapping)
	triggerMapping.AddFieldMappingsAt("Tags", keywordFieldMapping)
	triggerMapping.AddFieldMappingsAt("LastCheckScore", numericFieldMapping)

	indexMapping := bleve.NewIndexMapping()
	indexMapping.AddDocumentMapping(indexedTriggerCheck{}.Type(), triggerMapping)
	return indexMapping
}
