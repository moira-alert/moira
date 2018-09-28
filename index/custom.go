package index

import (
	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/mapping"
)

func buildCustomIndexMapping() (mapping.IndexMapping, error) {
	indexMapping := bleve.NewIndexMapping()

	var err error
	err = indexMapping.AddCustomCharFilter("url_links",
		map[string]interface{}{
			"regexp": `(https?:\/\/)?([\da-z\.-]+)\.([a-z\.]{2,6})([\/\w \.-]*)*`,
			"type":   `regexp`,
		})
	if err != nil {
		return nil, err
	}

	err = indexMapping.AddCustomAnalyzer("ENG+RUS, trim URLs",
		map[string]interface{}{
			"type": `custom`,
			"char_filters": []interface{}{
				`url_links`,
			},
			"tokenizer": `whitespace`,
			"token_filters": []interface{}{
				`to_lower`,
				`apostrophe`,
				`stop_en`,
				`stemmer_en`,
				`stop_ru`,
				`stemmer_ru`,
			},
		})
	if err != nil {
		return nil, err
	}

	return indexMapping, nil
}
