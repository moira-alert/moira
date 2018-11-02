package mapping

import (
	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/mapping"
)

type DocumentMapping interface {
	GetDocumentMapping() *mapping.DocumentMapping
	Type() string
}

func BuildIndexMapping(documents ...DocumentMapping) mapping.IndexMapping {
	indexMapping := bleve.NewIndexMapping()

	for _, document := range documents {
		documentMapping := document.GetDocumentMapping()
		indexMapping.AddDocumentMapping(document.Type(), documentMapping)
	}

	return indexMapping
}
