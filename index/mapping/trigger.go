package mapping

import (
	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/mapping"
	"github.com/moira-alert/moira"
)

// TriggerField is used as enum
type TriggerField int

// Constants used as enum
const (
	TriggerID TriggerField = iota
	TriggerName
	TriggerDesc
	TriggerTags
	TriggerLastCheckScore
)

var (
	triggerFieldNames = []string{
		"ID",
		"Name",
		"Desc",
		"Tags",
		"LastCheckScore",
	}
	triggerFieldTagValues = []string{
		"id",
		"name",
		"desc",
		"tags",
		"",
	}
)

// Trigger represents Moira.Trigger type for full-text search index. It includes only indexed fields
type Trigger struct {
	ID             string
	Name           string
	Desc           string
	Tags           []string
	LastCheckScore int64
}

// Type returns string with type name. It is used for Bleve.Search
func (Trigger) Type() string {
	return "moira.indexed.trigger"
}

// String returns TriggerField name. It works like enum
func (field TriggerField) String() string {
	return triggerFieldNames[field]
}

// GetTagValue returns TriggerField value used in marshalling. It works like enum
func (field TriggerField) GetTagValue() string {
	return triggerFieldTagValues[field]
}

// GetDocumentMapping returns Bleve.mapping.DocumentMapping for Trigger type
func (Trigger) GetDocumentMapping() *mapping.DocumentMapping {

	triggerMapping := bleve.NewDocumentStaticMapping()

	triggerMapping.AddFieldMappingsAt(TriggerName.String(), getStandardMapping())
	triggerMapping.AddFieldMappingsAt(TriggerTags.String(), getKeywordMapping())
	triggerMapping.AddFieldMappingsAt(TriggerDesc.String(), getStandardMapping())
	triggerMapping.AddFieldMappingsAt(TriggerLastCheckScore.String(), getNumericMapping())

	return triggerMapping

}

// CreateIndexedTrigger creates mapping.Trigger object out of moira.TriggerCheck
func CreateIndexedTrigger(triggerCheck *moira.TriggerCheck) Trigger {
	return Trigger{
		ID:             triggerCheck.ID,
		Name:           triggerCheck.Name,
		Desc:           moira.UseString(triggerCheck.Desc),
		Tags:           triggerCheck.Tags,
		LastCheckScore: triggerCheck.LastCheck.Score,
	}
}
