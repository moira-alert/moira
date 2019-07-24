package mapping

import (
	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/mapping"
	"github.com/moira-alert/moira"
)

// FieldData is container for field-related parameters
type FieldData struct {
	name     string
	nameTag  string
	priority float64
}

var (
	// TriggerID represents field data for moira.Trigger.ID
	TriggerID = FieldData{"ID", "id", 5}
	// TriggerName represents field data for moira.Trigger.Name
	TriggerName = FieldData{"Name", "name", 3}
	// TriggerDesc represents field data for moira.Trigger.Desc
	TriggerDesc = FieldData{"Desc", "desc", 1}
	// TriggerTags represents field data for moira.Trigger.Tags
	TriggerTags = FieldData{"Tags", "tags", 0}
	// TriggerLastCheckScore represents field data for moira.CheckData score
	TriggerLastCheckScore = FieldData{"LastCheckScore", "", 0}
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

// String returns TriggerField name.
func (field FieldData) String() string {
	return field.name
}

// GetTagValue returns TriggerField value used in marshalling.
func (field FieldData) GetTagValue() string {
	return field.nameTag
}

// GetPriority returns field priority
func (field FieldData) GetPriority() float64 {
	return field.priority
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
