package mapping

import (
	"log"
	"reflect"
	"strings"
	"testing"

	moira2 "github.com/moira-alert/moira/internal/moira"

	. "github.com/smartystreets/goconvey/convey"
)

var testTriggerFields = []FieldData{
	TriggerID,
	TriggerName,
	TriggerDesc,
	TriggerTags,
	TriggerLastCheckScore,
}

func TestTriggerField_GetPriority(t *testing.T) {
	expected := []float64{5, 3, 1, 0, 0}
	actual := make([]float64, 0, len(testTriggerFields))
	Convey("Test GetPriority returns correct field priority", t, func() {
		for _, triggerField := range testTriggerFields {
			fieldName, fieldPriority := triggerField.GetName(), triggerField.GetPriority()
			log.Printf("field: %s priority: %f", fieldName, fieldPriority)
			actual = append(actual, triggerField.GetPriority())
		}
		So(actual, ShouldResemble, expected)
	})
}

func TestTriggerField_GetTagValue(t *testing.T) {
	// This test is necessary to make sure that
	// SearchResult will contain highlights for actual moira.Trigger structure
	Convey("Test GetTagValue returns correct JSON tag", t, func() {
		for _, triggerField := range testTriggerFields {
			actual := getTagByFieldName(triggerField.GetName())
			expected := triggerField.GetTagValue()
			So(actual, ShouldEqual, expected)
		}
	})
}

// getTagByFieldName returns corresponding moira.Trigger JSON tag for given trigger field
func getTagByFieldName(fieldName string) string {
	var trigger moira2.Trigger
	var fieldTag string
	if field, ok := reflect.TypeOf(&trigger).Elem().FieldByName(fieldName); ok {
		fieldTag = field.Tag.Get("json")
		fieldTag = strings.Replace(fieldTag, ",omitempty", "", -1)
	}
	return fieldTag
}
