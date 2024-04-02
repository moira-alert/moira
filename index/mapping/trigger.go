package mapping

import (
	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/mapping"
	"github.com/moira-alert/moira"
)

var (
	// TriggerID represents field data for moira.Trigger.ID.
	TriggerID = FieldData{"ID", "id", 5}
	// TriggerName represents field data for moira.Trigger.Name.
	TriggerName = FieldData{"Name", "name", 3}
	// TriggerDesc represents field data for moira.Trigger.Desc.
	TriggerDesc = FieldData{"Desc", "desc", 1}
	// TriggerTags represents field data for moira.Trigger.Tags.
	TriggerTags = FieldData{"Tags", "tags", 0}
	// TriggerCreatedBy represents field data for moira.Trigger.CreatedBy.
	TriggerCreatedBy = FieldData{"CreatedBy", "created_by", 0}
	// TriggerLastCheckScore represents field data for moira.CheckData score.
	TriggerLastCheckScore = FieldData{"LastCheckScore", "", 0}
)

// Trigger represents Moira.Trigger type for full-text search index. It includes only indexed fields.
type Trigger struct {
	ID             string
	Name           string
	Desc           string
	Tags           []string
	CreatedBy      string
	LastCheckScore int64
}

// Type returns string with type name. It is used for Bleve.Search.
func (Trigger) Type() string {
	return "moira.indexed.trigger"
}

// GetDocumentMapping returns Bleve.mapping.DocumentMapping for Trigger type.
func (Trigger) GetDocumentMapping() *mapping.DocumentMapping {
	triggerMapping := bleve.NewDocumentStaticMapping()

	triggerMapping.AddFieldMappingsAt(TriggerID.GetName(), getKeywordMapping())
	triggerMapping.AddFieldMappingsAt(TriggerName.GetName(), getStandardMapping())
	triggerMapping.AddFieldMappingsAt(TriggerTags.GetName(), getKeywordMapping())
	triggerMapping.AddFieldMappingsAt(TriggerDesc.GetName(), getStandardMapping())
	triggerMapping.AddFieldMappingsAt(TriggerCreatedBy.GetName(), getKeywordMapping())
	triggerMapping.AddFieldMappingsAt(TriggerLastCheckScore.GetName(), getNumericMapping())

	return triggerMapping
}

// CreateIndexedTrigger creates mapping.Trigger object out of moira.TriggerCheck.
func CreateIndexedTrigger(triggerCheck *moira.TriggerCheck) Trigger {
	return Trigger{
		ID:             triggerCheck.ID,
		Name:           triggerCheck.Name,
		Desc:           moira.UseString(triggerCheck.Desc),
		Tags:           triggerCheck.Tags,
		CreatedBy:      triggerCheck.CreatedBy,
		LastCheckScore: triggerCheck.LastCheck.Score,
	}
}
