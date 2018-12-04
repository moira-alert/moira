package index

import (
	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/mapping"
)

func buildIndex(indexMapping mapping.IndexMapping) (bleve.Index, error) {

	bleveIdx, err := bleve.NewMemOnly(indexMapping)

	return bleveIdx, err
}
