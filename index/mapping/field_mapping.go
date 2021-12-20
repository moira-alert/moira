package mapping

import (
	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/keyword"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/standard"
	"github.com/blevesearch/bleve/v2/mapping"
)

func getKeywordMapping() *mapping.FieldMapping {
	keywordFieldMapping := bleve.NewTextFieldMapping()
	keywordFieldMapping.Analyzer = keyword.Name
	keywordFieldMapping.Store = false
	keywordFieldMapping.IncludeTermVectors = false
	keywordFieldMapping.IncludeInAll = false

	return keywordFieldMapping
}

func getStandardMapping() *mapping.FieldMapping {
	standardFieldMapping := bleve.NewTextFieldMapping()
	standardFieldMapping.Analyzer = standard.Name
	standardFieldMapping.Store = true
	standardFieldMapping.IncludeTermVectors = true

	return standardFieldMapping
}

func getNumericMapping() *mapping.FieldMapping {
	return bleve.NewNumericFieldMapping()
}
