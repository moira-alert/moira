package index

import "github.com/moira-alert/moira"

type indexedTriggerCheck struct {
	ID             string
	Name           string
	Desc           string
	Tags           []string
	LastCheckScore int64
}

// Type returns string with type name. It is used for Bleve.Search
func (indexedTriggerCheck) Type() string {
	return "moira.trigger"
}

func createIndexedTriggerCheck(triggerCheck moira.TriggerCheck) indexedTriggerCheck {
	return indexedTriggerCheck{
		ID:             triggerCheck.ID,
		Name:           triggerCheck.Name,
		Desc:           moira.UseString(triggerCheck.Desc),
		Tags:           triggerCheck.Tags,
		LastCheckScore: triggerCheck.LastCheck.Score,
	}
}
