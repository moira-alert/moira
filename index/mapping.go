package index

import (
	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/analysis/analyzer/keyword"
	"github.com/blevesearch/bleve/analysis/analyzer/standard"
	"github.com/blevesearch/bleve/mapping"
	"github.com/moira-alert/moira"
)

func buildIndexMapping() mapping.IndexMapping {

	// a generic reusable mapping for english text
	standardFieldMapping := bleve.NewTextFieldMapping()
	standardFieldMapping.Analyzer = standard.Name

	// a generic reusable mapping for keyword text
	keywordFieldMapping := bleve.NewTextFieldMapping()
	keywordFieldMapping.Analyzer = keyword.Name

	// the static document mapping will only index the fields you explicitly configure and will not "dynamically" try to invent mappings for all fields it finds
	triggerMapping := bleve.NewDocumentStaticMapping()

	triggerMapping.AddFieldMappingsAt("name", standardFieldMapping)
	triggerMapping.AddFieldMappingsAt("description", standardFieldMapping)
	triggerMapping.AddFieldMappingsAt("tags", keywordFieldMapping)

	indexMapping := bleve.NewIndexMapping()
	indexMapping.AddDocumentMapping(moira.Trigger{}.Type(), triggerMapping)
	return indexMapping
}
