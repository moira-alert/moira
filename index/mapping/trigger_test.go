package mapping

import (
	"reflect"
	"strings"
	"testing"

	"github.com/moira-alert/moira"
	. "github.com/smartystreets/goconvey/convey"
)

func TestTriggerField_GetTagValue(t *testing.T) {
	// This test is necessary to make sure that
	// SearchResult will contain highlights for actual moira.Trigger structure
	triggerFields := []TriggerField{
		TriggerID,
		TriggerName,
		TriggerDesc,
		TriggerTags,
		TriggerLastCheckScore,
	}
	Convey("Test GetTagValue returns correct JSON tag", t, func(c C) {
		for _, triggerField := range triggerFields {
			actual := getTagByFieldName(triggerField.String())
			expected := triggerField.GetTagValue()
			c.So(actual, ShouldEqual, expected)
		}
	})
}

// getTagByFieldName returns corresponding moira.Trigger JSON tag for given trigger field
func getTagByFieldName(fieldName string) string {
	var trigger moira.Trigger
	var fieldTag string
	if field, ok := reflect.TypeOf(&trigger).Elem().FieldByName(fieldName); ok {
		fieldTag = field.Tag.Get("json")
		fieldTag = strings.Replace(fieldTag, ",omitempty", "", -1)
	}
	return fieldTag
}
