package mapping

import (
	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/mapping"
	"github.com/moira-alert/moira"
)

type TriggerField int

const (
	TriggerID             TriggerField = 0
	TriggerName           TriggerField = 1
	TriggerDesc           TriggerField = 2
	TriggerTags           TriggerField = 3
	TriggerLastCheckScore TriggerField = 4
)

var triggerFieldNames = []string{"ID", "Name", "Desc", "Tags", "LastCheckScore"}

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

// GetDocumentMapping returns Bleve.mapping.DocumentMapping for Trigger type
func (Trigger) GetDocumentMapping() *mapping.DocumentMapping {

	triggerMapping := bleve.NewDocumentStaticMapping()

	triggerMapping.AddFieldMappingsAt(TriggerName.String(), getStandardMapping())
	triggerMapping.AddFieldMappingsAt(TriggerTags.String(), getKeywordMapping())
	// ToDo: do we want to index description?
	//triggerMapping.AddFieldMappingsAt(TriggerDesc.String(), getStandardMapping())
	triggerMapping.AddFieldMappingsAt(TriggerLastCheckScore.String(), getNumericMapping())

	return triggerMapping

}

// CreateIndexedTrigger creates mapping.Trigger object out of moira.TriggerCheck
func CreateIndexedTrigger(triggerCheck moira.TriggerCheck) Trigger {
	return Trigger{
		ID:             triggerCheck.ID,
		Name:           triggerCheck.Name,
		Desc:           moira.UseString(triggerCheck.Desc),
		Tags:           triggerCheck.Tags,
		LastCheckScore: triggerCheck.LastCheck.Score,
	}
}
