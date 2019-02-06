package controller

import (
	"fmt"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/dto"
	"github.com/moira-alert/moira/mock/moira-alert"
	"github.com/op/go-logging"
	. "github.com/smartystreets/goconvey/convey"
)

func TestDeletePattern(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)

	Convey("Success", t, func() {
		dataBase.EXPECT().RemovePattern("super.puper.pattern").Return(nil)
		err := DeletePattern(dataBase, "super.puper.pattern")
		So(err, ShouldBeNil)
	})

	Convey("Error", t, func() {
		expected := fmt.Errorf("oooops! Can not remove pattern")
		dataBase.EXPECT().RemovePattern("super.puper.pattern").Return(expected)
		err := DeletePattern(dataBase, "super.puper.pattern")
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
	})
}

func TestGetAllPatterns(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
	logger, _ := logging.GetLogger("Test")
	defer mockCtrl.Finish()
	pattern1 := "my.first.pattern"
	pattern2 := "my.second.pattern"

	Convey("One pattern more triggers", t, func() {
		triggers := []*dto.TriggerModel{{ID: uuid.Must(uuid.NewV4()).String()}, {ID: uuid.Must(uuid.NewV4()).String()}}
		metrics := []string{"my.first.metric"}
		dataBase.EXPECT().GetPatterns().Return([]string{pattern1}, nil)
		expectGettingPatternList(dataBase, pattern1, triggers, metrics)
		list, err := GetAllPatterns(dataBase, logger)
		So(err, ShouldBeNil)
		So(list, ShouldResemble, &dto.PatternList{
			List: []dto.PatternData{{Metrics: metrics, Pattern: pattern1, Triggers: []dto.TriggerModel{*triggers[0], *triggers[1]}}},
		})
	})

	Convey("Many patterns one trigger", t, func() {
		triggers1 := []*dto.TriggerModel{{ID: "1111"}, {ID: "111111"}}
		triggers2 := []*dto.TriggerModel{{ID: "22222"}}
		metrics1 := []string{"my.first.metric"}
		metrics2 := []string{"my.second.metric"}
		dataBase.EXPECT().GetPatterns().Return([]string{pattern1, pattern2}, nil)
		expectGettingPatternList(dataBase, pattern1, triggers1, metrics1)
		expectGettingPatternList(dataBase, pattern2, triggers2, metrics2)
		list, err := GetAllPatterns(dataBase, logger)
		So(err, ShouldBeNil)
		So(list.List, ShouldHaveLength, 2)
		for _, patternStat := range list.List {
			if patternStat.Pattern == pattern1 {
				So(patternStat, ShouldResemble, dto.PatternData{Metrics: metrics1, Pattern: pattern1, Triggers: []dto.TriggerModel{*triggers1[0], *triggers1[1]}})
			}
			if patternStat.Pattern == pattern2 {
				So(patternStat, ShouldResemble, dto.PatternData{Metrics: metrics2, Pattern: pattern2, Triggers: []dto.TriggerModel{*triggers2[0]}})
			}
		}
	})

	Convey("Test errors", t, func() {
		Convey("GetPatterns error", func() {
			expected := fmt.Errorf("oh no!!!11 Cant get patterns")
			dataBase.EXPECT().GetPatterns().Return(nil, expected)
			list, err := GetAllPatterns(dataBase, logger)
			So(err, ShouldResemble, api.ErrorInternalServer(expected))
			So(list, ShouldBeNil)
		})
	})
}

func expectGettingPatternList(database *mock_moira_alert.MockDatabase, pattern string, triggers []*dto.TriggerModel, metrics []string) {
	tr := make([]*moira.Trigger, 0)
	for _, trigger := range triggers {
		tr = append(tr, trigger.ToMoiraTrigger())
	}

	database.EXPECT().GetPatternTriggerIDs(pattern).Return([]string{pattern}, nil)
	database.EXPECT().GetTriggers([]string{pattern}).Return(tr, nil)
	database.EXPECT().GetPatternMetrics(pattern).Return(metrics, nil)
}
