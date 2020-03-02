package mapping

import (
	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/mapping"
)

// DocumentMapping implements mapping.DocumentMapping functionality
type DocumentMapping interface {
	GetDocumentMapping() *mapping.DocumentMapping
	Type() string
}

// BuildIndexMapping gets slice of documents (DocumentMapping interface) and returns index with those documents mappings
func BuildIndexMapping(documents ...DocumentMapping) mapping.IndexMapping {
	indexMapping := bleve.NewIndexMapping()

	for _, document := range documents {
		documentMapping := document.GetDocumentMapping()
		indexMapping.AddDocumentMapping(document.Type(), documentMapping)
	}

	return indexMapping
}
